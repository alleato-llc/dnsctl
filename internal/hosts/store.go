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
// document. The write is durable against partial writes: content goes to a
// temp file in the same directory and is renamed into place.
func (s *Store) Save(doc *Document) error {
	mode := os.FileMode(0644)
	if fi, err := os.Stat(s.Path); err == nil {
		mode = fi.Mode().Perm()
		if err := s.backup(); err != nil {
			return err
		}
	}

	dir := filepath.Dir(s.Path)
	tmp, err := os.CreateTemp(dir, ".dnsctl-hosts-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once renamed

	if _, err := tmp.Write(doc.Render()); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpName, s.Path); err != nil {
		return fmt.Errorf("replace %s: %w", s.Path, err)
	}
	return nil
}

// backup copies the current file to "<path>.dnsctl.bak", overwriting any
// previous backup.
func (s *Store) backup() error {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return fmt.Errorf("read for backup: %w", err)
	}
	bak := s.Path + ".dnsctl.bak"
	if err := os.WriteFile(bak, data, 0644); err != nil {
		return fmt.Errorf("write backup %s: %w", bak, err)
	}
	return nil
}
