package hosts

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultPath is the system hosts file edited when no path is given.
const DefaultPath = "/etc/hosts"

// Store reads and writes a hosts file at a fixed path.
type Store struct {
	Path string
}

// NewStore returns a Store for the given path, defaulting to DefaultPath when
// path is empty.
func NewStore(path string) *Store {
	if path == "" {
		path = DefaultPath
	}
	return &Store{Path: path}
}

// Load reads and parses the hosts file. A missing file parses as empty so the
// managed block can be created on first write.
func (s *Store) Load() (*Document, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return Parse(nil), nil
		}
		return nil, fmt.Errorf("read %s: %w", s.Path, err)
	}
	return Parse(data), nil
}

// Save backs up the current file then atomically replaces it with the rendered
// document. It is a convenience wrapper over WriteAtomic.
func (s *Store) Save(doc *Document) error {
	return WriteAtomic(s.Path, doc.Render())
}

// WriteAtomic backs up the file at path (when it exists) then atomically
// replaces it with content. Content is written to a temp file in the same
// directory and renamed into place, so an interrupted write cannot corrupt the
// target; the original file mode is preserved.
//
// This is the single privileged side effect for hosts editing: writing a
// root-owned file such as /etc/hosts requires the calling process to be root,
// so it is invoked through a PrivilegedRunner (see internal/service).
func WriteAtomic(path string, content []byte) error {
	mode := os.FileMode(0644)
	if fi, err := os.Stat(path); err == nil {
		mode = fi.Mode().Perm()
		if err := backup(path); err != nil {
			return err
		}
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".dnsctl-hosts-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}

// backup copies the file at path to "<path>.dnsctl.bak", overwriting any
// previous backup.
func backup(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read for backup: %w", err)
	}
	bak := path + ".dnsctl.bak"
	if err := os.WriteFile(bak, data, 0644); err != nil {
		return fmt.Errorf("write backup %s: %w", bak, err)
	}
	return nil
}
