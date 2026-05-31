package service

import (
	"sync"

	"github.com/nycjv321/dnsctl/internal/dns"
	"github.com/nycjv321/dnsctl/internal/hosts"
)

// PrivilegedRunner performs the operations that require root: changing resolver
// configuration, flushing the DNS cache, and writing the hosts file.
//
// Frontends select an implementation rather than calling these directly:
//   - DirectRunner runs in-process (used when the process is already root, e.g.
//     `sudo dnsctl`, and inside the privileged helper daemon).
//   - HelperClient forwards each operation to the root helper over IPC (used by
//     the unprivileged GUI and a non-root CLI).
//
// This is the single seam where privilege is acquired; everything above it
// (the *Service types) is privilege-agnostic and reusable by CLI, TUI, and GUI.
type PrivilegedRunner interface {
	SetDNS(service string, servers []string) error
	ClearDNS(service string) error
	FlushDNS() error
	SaveHosts(path string, content []byte) error
}

// DirectRunner performs privileged operations in the current process. It only
// succeeds when that process actually has the necessary privileges.
type DirectRunner struct {
	mu     sync.Mutex
	client dns.Client
}

var _ PrivilegedRunner = (*DirectRunner)(nil)

// NewDirectRunner returns a DirectRunner. The DNS client is created lazily, so
// hosts operations work even on platforms without a supported DNS backend.
func NewDirectRunner() *DirectRunner {
	return &DirectRunner{}
}

// NewDirectRunnerWithClient returns a DirectRunner that uses the given DNS
// client instead of creating one lazily. Useful for sharing a single client
// with a ResolverService, and for injecting a mock in tests.
func NewDirectRunnerWithClient(client dns.Client) *DirectRunner {
	return &DirectRunner{client: client}
}

func (r *DirectRunner) ensureClient() (dns.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.client == nil {
		c, err := dns.NewClient()
		if err != nil {
			return nil, err
		}
		r.client = c
	}
	return r.client, nil
}

func (r *DirectRunner) SetDNS(service string, servers []string) error {
	c, err := r.ensureClient()
	if err != nil {
		return err
	}
	return c.SetDNSServers(service, servers)
}

func (r *DirectRunner) ClearDNS(service string) error {
	c, err := r.ensureClient()
	if err != nil {
		return err
	}
	return c.ClearDNSServers(service)
}

func (r *DirectRunner) FlushDNS() error {
	c, err := r.ensureClient()
	if err != nil {
		return err
	}
	return c.FlushCache()
}

func (r *DirectRunner) SaveHosts(path string, content []byte) error {
	return hosts.WriteAtomic(path, content)
}
