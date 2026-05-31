package service

import "github.com/nycjv321/dnsctl/internal/dns"

// ResolverService manages the system resolver configuration (which DNS servers
// a network service uses). Reads go straight to the DNS client; writes go
// through the PrivilegedRunner, mirroring HostsService.
type ResolverService struct {
	client dns.Client
	runner PrivilegedRunner
}

// NewResolverService builds the resolver facade. client is used for
// (unprivileged) reads; runner performs the (privileged) writes.
func NewResolverService(client dns.Client, runner PrivilegedRunner) *ResolverService {
	return &ResolverService{client: client, runner: runner}
}

// ListServices returns the available network services/interfaces.
func (s *ResolverService) ListServices() ([]string, error) {
	return s.client.ListNetworkServices()
}

// CurrentDNS returns the DNS servers currently set for a network service.
func (s *ResolverService) CurrentDNS(service string) ([]string, error) {
	return s.client.GetDNSServers(service)
}

// Backend returns the underlying DNS backend's display name.
func (s *ResolverService) Backend() string {
	return s.client.Name()
}

// primaryServiceProvider is implemented by DNS clients that can report the
// active (default-route) network service. Backends that can't simply don't
// implement it.
type primaryServiceProvider interface {
	PrimaryService() (string, error)
}

// PrimaryService returns the active/default network service, or "" if the
// backend can't determine one.
func (s *ResolverService) PrimaryService() (string, error) {
	if p, ok := s.client.(primaryServiceProvider); ok {
		return p.PrimaryService()
	}
	return "", nil
}

// Set applies specific DNS servers to a network service.
func (s *ResolverService) Set(service string, servers []string) error {
	return s.runner.SetDNS(service, servers)
}

// Clear reverts a network service to DHCP-provided DNS.
func (s *ResolverService) Clear(service string) error {
	return s.runner.ClearDNS(service)
}

// Flush flushes the DNS cache. It is best-effort at call sites that treat
// flushing as non-fatal.
func (s *ResolverService) Flush() error {
	return s.runner.FlushDNS()
}
