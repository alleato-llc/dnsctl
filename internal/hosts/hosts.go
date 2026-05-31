// Package hosts provides CRUD operations over a managed block of entries in
// an /etc/hosts-style file.
//
// dnsctl never edits lines outside its own managed block, which is delimited
// by sentinel comments. Everything before and after the block is preserved
// byte-for-byte, so system entries (localhost, broadcasthost, ...) and any
// hand-added lines are never touched.
package hosts

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Sentinels delimiting the dnsctl-managed block. Matching is done by prefix so
// the human-readable suffix can change without breaking existing files.
const (
	beginPrefix = "# BEGIN dnsctl"
	endPrefix   = "# END dnsctl"

	beginMarker = "# BEGIN dnsctl (managed by `dnsctl hosts` — do not edit by hand)"
	endMarker   = "# END dnsctl"
)

// Entry is a single host mapping: one IP, a primary hostname, optional aliases
// sharing the same line, and an optional trailing comment.
type Entry struct {
	IP       string   `json:"ip"`
	Hostname string   `json:"hostname"`
	Aliases  []string `json:"aliases,omitempty"`
	Comment  string   `json:"comment,omitempty"`
}

// hostnameRe is a permissive RFC-1123-ish hostname check: dot-separated labels
// of letters, digits and hyphens (not leading/trailing a label).
var hostnameRe = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// Validate reports the first problem with an entry, or nil if it is usable.
func (e Entry) Validate() error {
	if net.ParseIP(e.IP) == nil {
		return fmt.Errorf("invalid IP address: %q", e.IP)
	}
	if !validHostname(e.Hostname) {
		return fmt.Errorf("invalid hostname: %q", e.Hostname)
	}
	for _, a := range e.Aliases {
		if !validHostname(a) {
			return fmt.Errorf("invalid alias: %q", a)
		}
	}
	if strings.ContainsRune(e.Comment, '\n') {
		return fmt.Errorf("comment may not contain newlines")
	}
	return nil
}

func validHostname(h string) bool {
	return h != "" && len(h) <= 253 && hostnameRe.MatchString(h)
}

// render produces the canonical single-line form of an entry.
func (e Entry) render() string {
	names := e.Hostname
	if len(e.Aliases) > 0 {
		names += " " + strings.Join(e.Aliases, " ")
	}
	line := e.IP + "\t" + names
	if e.Comment != "" {
		line += "\t# " + e.Comment
	}
	return line
}

// Document is a parsed hosts file split into the content surrounding the
// managed block and the structured entries inside it.
type Document struct {
	head     string  // verbatim content before the managed block
	tail     string  // verbatim content after the managed block
	hasBlock bool    // whether a managed block was present in the source
	entries  []Entry // managed entries, in file order
}

// Parse splits raw hosts-file content into surrounding text and managed entries.
func Parse(content []byte) *Document {
	lines := strings.Split(string(content), "\n")

	begin, end := -1, -1
	for i, ln := range lines {
		if begin == -1 && strings.HasPrefix(strings.TrimSpace(ln), beginPrefix) {
			begin = i
			continue
		}
		if begin != -1 && strings.HasPrefix(strings.TrimSpace(ln), endPrefix) {
			end = i
			break
		}
	}

	// No well-formed block: treat the whole file as head.
	if begin == -1 || end == -1 {
		return &Document{head: string(content)}
	}

	doc := &Document{hasBlock: true}
	doc.head = strings.Join(lines[:begin], "\n")
	doc.tail = strings.Join(lines[end+1:], "\n")
	for _, ln := range lines[begin+1 : end] {
		if e, ok := parseEntryLine(ln); ok {
			doc.entries = append(doc.entries, e)
		}
	}
	return doc
}

// parseEntryLine parses one managed line into an Entry. Blank and comment-only
// lines return ok=false (the block is canonicalised on write, so stray lines
// inside it are dropped).
func parseEntryLine(line string) (Entry, bool) {
	var comment string
	if i := strings.IndexByte(line, '#'); i >= 0 {
		comment = strings.TrimSpace(line[i+1:])
		line = line[:i]
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return Entry{}, false
	}
	return Entry{
		IP:       fields[0],
		Hostname: fields[1],
		Aliases:  append([]string(nil), fields[2:]...),
		Comment:  comment,
	}, true
}

// List returns a copy of the managed entries in file order. The result is
// always non-nil (an empty list marshals to a JSON array, not null).
func (d *Document) List() []Entry {
	return append(make([]Entry, 0, len(d.entries)), d.entries...)
}

// Get returns the managed entry for a hostname (case-insensitive).
func (d *Document) Get(hostname string) (Entry, bool) {
	if i := d.indexOf(hostname); i >= 0 {
		return d.entries[i], true
	}
	return Entry{}, false
}

// Set upserts an entry, keyed on hostname (case-insensitive). It returns true
// if an existing entry was replaced, false if a new one was appended.
func (d *Document) Set(e Entry) bool {
	if i := d.indexOf(e.Hostname); i >= 0 {
		d.entries[i] = e
		return true
	}
	d.entries = append(d.entries, e)
	return false
}

// Remove deletes the entry for a hostname (case-insensitive), reporting whether
// anything was removed.
func (d *Document) Remove(hostname string) bool {
	i := d.indexOf(hostname)
	if i < 0 {
		return false
	}
	d.entries = append(d.entries[:i], d.entries[i+1:]...)
	return true
}

func (d *Document) indexOf(hostname string) int {
	for i, e := range d.entries {
		if strings.EqualFold(e.Hostname, hostname) {
			return i
		}
	}
	return -1
}

// Render serialises the document back to hosts-file content, rewriting the
// managed block from the structured entries and preserving everything else.
// When there are no managed entries, the block is omitted entirely.
func (d *Document) Render() []byte {
	head := strings.TrimRight(d.head, "\n")
	tail := strings.TrimLeft(d.tail, "\n")

	var b strings.Builder
	if head != "" {
		b.WriteString(head)
		b.WriteString("\n")
	}

	if len(d.entries) > 0 {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(beginMarker)
		b.WriteString("\n")
		for _, e := range d.entries {
			b.WriteString(e.render())
			b.WriteString("\n")
		}
		b.WriteString(endMarker)
		b.WriteString("\n")
	}

	if tail != "" {
		b.WriteString(tail)
		b.WriteString("\n")
	}

	return []byte(b.String())
}
