// Package service is the privilege-agnostic facade that CLI, TUI, and GUI all
// build on. It orchestrates the domain packages (config, dns, hosts) and routes
// the operations that need root through a PrivilegedRunner.
//
// The exported types here are intended to double as the binding surface for the
// Wails (Go -> TypeScript) GUI, so they use plain fields and JSON tags.
//
// A ResolverService for the DNS-profile side will follow the same shape
// (read via the dns client, mutate via the runner's SetDNS/ClearDNS/FlushDNS).
package service

import (
	"errors"
	"fmt"

	"github.com/nycjv321/dnsctl/internal/hosts"
)

// Sentinel errors so callers (CLI exit codes, GUI dialogs) can branch on the
// failure mode rather than parsing strings.
var (
	ErrExists   = errors.New("entry already exists")
	ErrNotFound = errors.New("no managed entry")
)

// ApplyOptions modifies how a mutating operation is carried out.
type ApplyOptions struct {
	// DryRun computes the resulting file but does not write it.
	DryRun bool `json:"dryRun"`
	// Flush flushes the DNS cache after a successful write.
	Flush bool `json:"flush"`
}

// HostsService manages dnsctl's /etc/hosts entries. Reads go straight to the
// file; writes go through the PrivilegedRunner.
type HostsService struct {
	store  *hosts.Store
	runner PrivilegedRunner
}

// NewHostsService returns a HostsService for the given hosts file path (empty
// means hosts.DefaultPath), persisting changes through runner.
func NewHostsService(path string, runner PrivilegedRunner) *HostsService {
	return &HostsService{store: hosts.NewStore(path), runner: runner}
}

// List returns the managed entries.
func (s *HostsService) List() ([]hosts.Entry, error) {
	doc, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	return doc.List(), nil
}

// Get returns the managed entry for a hostname, if present.
func (s *HostsService) Get(hostname string) (hosts.Entry, bool, error) {
	doc, err := s.store.Load()
	if err != nil {
		return hosts.Entry{}, false, err
	}
	e, ok := doc.Get(hostname)
	return e, ok, nil
}

// Add inserts a new entry, failing with ErrExists if the hostname is already
// managed.
func (s *HostsService) Add(e hosts.Entry, opts ApplyOptions) ([]byte, error) {
	return s.mutate(opts, func(doc *hosts.Document) error {
		if _, exists := doc.Get(e.Hostname); exists {
			return fmt.Errorf("%w: %q", ErrExists, e.Hostname)
		}
		if err := e.Validate(); err != nil {
			return err
		}
		doc.Set(e)
		return nil
	})
}

// Set upserts an entry (idempotent).
func (s *HostsService) Set(e hosts.Entry, opts ApplyOptions) ([]byte, error) {
	return s.mutate(opts, func(doc *hosts.Document) error {
		if err := e.Validate(); err != nil {
			return err
		}
		doc.Set(e)
		return nil
	})
}

// Remove deletes an entry, failing with ErrNotFound if it is not managed.
func (s *HostsService) Remove(hostname string, opts ApplyOptions) ([]byte, error) {
	return s.mutate(opts, func(doc *hosts.Document) error {
		if !doc.Remove(hostname) {
			return fmt.Errorf("%w: %q", ErrNotFound, hostname)
		}
		return nil
	})
}

// mutate loads the document, applies fn, then either returns the rendered
// result (DryRun) or persists it via the runner and optionally flushes the DNS
// cache. The rendered content is always returned so callers can preview it.
func (s *HostsService) mutate(opts ApplyOptions, fn func(*hosts.Document) error) ([]byte, error) {
	doc, err := s.store.Load()
	if err != nil {
		return nil, err
	}
	if err := fn(doc); err != nil {
		return nil, err
	}
	content := doc.Render()
	if opts.DryRun {
		return content, nil
	}
	if err := s.runner.SaveHosts(s.store.Path, content); err != nil {
		return nil, err
	}
	if opts.Flush {
		if err := s.runner.FlushDNS(); err != nil {
			return content, fmt.Errorf("flush: %w", err)
		}
	}
	return content, nil
}
