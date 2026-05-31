package hosts

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

const systemHosts = `##
# Host Database
#
127.0.0.1	localhost
255.255.255.255	broadcasthost
::1		localhost
`

func TestParse_NoBlock(t *testing.T) {
	doc := Parse([]byte(systemHosts))
	if doc.hasBlock {
		t.Fatal("expected no managed block")
	}
	if len(doc.List()) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(doc.List()))
	}
}

func TestParse_WithBlock(t *testing.T) {
	content := systemHosts + `
# BEGIN dnsctl (managed)
10.0.0.5	staging.api	api2.local	# staging
127.0.0.1	myapp.local
# END dnsctl
`
	doc := Parse([]byte(content))
	if !doc.hasBlock {
		t.Fatal("expected a managed block")
	}
	entries := doc.List()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].IP != "10.0.0.5" || entries[0].Hostname != "staging.api" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if len(entries[0].Aliases) != 1 || entries[0].Aliases[0] != "api2.local" {
		t.Errorf("unexpected aliases: %+v", entries[0].Aliases)
	}
	if entries[0].Comment != "staging" {
		t.Errorf("unexpected comment: %q", entries[0].Comment)
	}
}

func TestList_EmptyMarshalsToJSONArray(t *testing.T) {
	doc := Parse([]byte(systemHosts)) // no managed block -> no managed entries

	list := doc.List()
	if list == nil {
		t.Fatal("List() must be non-nil so it marshals to [] not null")
	}
	data, err := json.Marshal(list)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(data) != "[]" {
		t.Errorf("empty list marshalled to %q, want %q", data, "[]")
	}
}

func TestUnmanaged(t *testing.T) {
	content := systemHosts + `
# BEGIN dnsctl (managed)
10.0.0.5	staging.api
# END dnsctl
`
	doc := Parse([]byte(content))

	sys := doc.Unmanaged()
	names := map[string]bool{}
	for _, e := range sys {
		names[e.Hostname] = true
	}
	if !names["localhost"] || !names["broadcasthost"] {
		t.Errorf("expected system entries in Unmanaged(), got %+v", sys)
	}
	if names["staging.api"] {
		t.Error("managed entry must not appear in Unmanaged()")
	}
	// Always non-nil so it marshals to [].
	if Parse(nil).Unmanaged() == nil {
		t.Error("Unmanaged() must be non-nil")
	}
}

func TestSet_AddAndUpdate(t *testing.T) {
	doc := Parse(nil)

	if replaced := doc.Set(Entry{IP: "1.2.3.4", Hostname: "a.local"}); replaced {
		t.Error("first Set should append, not replace")
	}
	if replaced := doc.Set(Entry{IP: "5.6.7.8", Hostname: "A.LOCAL"}); !replaced {
		t.Error("Set with same hostname (different case) should replace")
	}
	if len(doc.List()) != 1 {
		t.Fatalf("expected 1 entry after upsert, got %d", len(doc.List()))
	}
	got, _ := doc.Get("a.local")
	if got.IP != "5.6.7.8" {
		t.Errorf("expected updated IP 5.6.7.8, got %s", got.IP)
	}
}

func TestRemove(t *testing.T) {
	doc := Parse(nil)
	doc.Set(Entry{IP: "1.2.3.4", Hostname: "a.local"})

	if !doc.Remove("A.LOCAL") {
		t.Error("Remove should be case-insensitive and report success")
	}
	if doc.Remove("missing.local") {
		t.Error("Remove of missing host should report false")
	}
	if len(doc.List()) != 0 {
		t.Errorf("expected empty after remove, got %d", len(doc.List()))
	}
}

func TestRender_PreservesSurroundingContent(t *testing.T) {
	doc := Parse([]byte(systemHosts))
	doc.Set(Entry{IP: "127.0.0.1", Hostname: "myapp.local", Comment: "dev"})

	out := string(doc.Render())
	if !strings.Contains(out, "127.0.0.1\tlocalhost") {
		t.Error("system localhost entry was lost")
	}
	if !strings.Contains(out, "broadcasthost") {
		t.Error("broadcasthost entry was lost")
	}
	if !strings.Contains(out, beginMarker) || !strings.Contains(out, endMarker) {
		t.Error("managed block markers missing")
	}
	if !strings.Contains(out, "127.0.0.1\tmyapp.local\t# dev") {
		t.Errorf("managed entry not rendered as expected:\n%s", out)
	}
}

func TestRender_RoundTrip(t *testing.T) {
	doc := Parse([]byte(systemHosts))
	doc.Set(Entry{IP: "10.0.0.5", Hostname: "staging.api", Aliases: []string{"api2.local"}})

	reparsed := Parse(doc.Render())
	got := reparsed.List()
	if len(got) != 1 || got[0].Hostname != "staging.api" || len(got[0].Aliases) != 1 {
		t.Fatalf("round-trip lost data: %+v", got)
	}
}

func TestRender_EmptyBlockOmitted(t *testing.T) {
	doc := Parse([]byte(systemHosts))
	out := string(doc.Render())
	if strings.Contains(out, beginPrefix) {
		t.Error("empty managed block should not be written")
	}
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name  string
		entry Entry
		ok    bool
	}{
		{"valid v4", Entry{IP: "1.2.3.4", Hostname: "a.local"}, true},
		{"valid v6", Entry{IP: "::1", Hostname: "a.local"}, true},
		{"bad ip", Entry{IP: "999.1.1.1", Hostname: "a.local"}, false},
		{"bad host", Entry{IP: "1.2.3.4", Hostname: "no_underscores"}, false},
		{"bad alias", Entry{IP: "1.2.3.4", Hostname: "a.local", Aliases: []string{"bad host"}}, false},
		{"comment newline", Entry{IP: "1.2.3.4", Hostname: "a.local", Comment: "a\nb"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.entry.Validate()
			if (err == nil) != c.ok {
				t.Errorf("Validate()=%v, want ok=%v", err, c.ok)
			}
		})
	}
}

func TestStore_SaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts")
	store := NewStore(path)

	doc, err := store.Load()
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	doc.Set(Entry{IP: "1.2.3.4", Hostname: "a.local"})
	if err := store.Save(doc); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := store.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if _, ok := reloaded.Get("a.local"); !ok {
		t.Error("entry did not survive save/load")
	}
}

func TestStore_BackupCreatedOnOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hosts")
	store := NewStore(path)

	doc := Parse([]byte(systemHosts))
	if err := store.Save(doc); err != nil {
		t.Fatalf("first save: %v", err)
	}
	// Second save should back up the prior content.
	doc.Set(Entry{IP: "1.2.3.4", Hostname: "a.local"})
	if err := store.Save(doc); err != nil {
		t.Fatalf("second save: %v", err)
	}

	if _, err := NewStore(path + ".dnsctl.bak").Load(); err != nil {
		t.Errorf("expected backup file: %v", err)
	}
}
