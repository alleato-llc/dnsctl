// Package guiapi is the binding layer between a GUI frontend (e.g. Wails,
// Go -> TypeScript) and dnsctl's service facade. The App methods are the surface
// the frontend calls; they delegate to HostsService/ResolverService and never
// touch privilege directly.
//
// It lives in the main module (not internal/) so the separate Wails module
// under gui/ can import it. Because it has no GUI-framework dependency, it
// compiles and is unit-tested as part of the normal build.
package guiapi

import (
	"context"

	"github.com/nycjv321/dnsctl/internal/dns"
	"github.com/nycjv321/dnsctl/internal/hosts"
	"github.com/nycjv321/dnsctl/internal/service"
)

// App is the bound object exposed to the frontend.
type App struct {
	ctx         context.Context
	hosts       *service.HostsService
	resolver    *service.ResolverService
	resolverErr error // set when no DNS backend is available; resolver methods report it
}

// New builds an App from already-constructed services (used in tests and by
// callers that want to choose the runner).
func New(hostsSvc *service.HostsService, resolver *service.ResolverService) *App {
	return &App{hosts: hostsSvc, resolver: resolver}
}

// NewApp builds the production App: privileged operations are forwarded to the
// dnsctl-helper (the GUI runs unprivileged), hosts editing targets /etc/hosts,
// and resolver reads use the platform DNS client when one is available.
func NewApp() *App {
	runner := service.NewHelperClient()
	app := &App{
		hosts: service.NewHostsService(hosts.DefaultPath, runner),
	}
	client, err := dns.NewClient()
	if err != nil {
		app.resolverErr = err
	} else {
		app.resolver = service.NewResolverService(client, runner)
	}
	return app
}

// Startup is the Wails startup hook; it captures the app context.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// --- Hosts ---

// ListHosts returns the dnsctl-managed /etc/hosts entries.
func (a *App) ListHosts() ([]hosts.Entry, error) {
	return a.hosts.List()
}

// AddHost adds a new entry, failing if the hostname already exists.
func (a *App) AddHost(entry hosts.Entry) error {
	_, err := a.hosts.Add(entry, service.ApplyOptions{})
	return err
}

// SetHost adds or updates an entry (idempotent).
func (a *App) SetHost(entry hosts.Entry) error {
	_, err := a.hosts.Set(entry, service.ApplyOptions{})
	return err
}

// RemoveHost deletes a managed entry by hostname.
func (a *App) RemoveHost(hostname string) error {
	_, err := a.hosts.Remove(hostname, service.ApplyOptions{})
	return err
}

// --- Resolver (read-only status) ---

// ServiceDNS is the current resolver configuration for one network service.
// dnsctl does not modify this; it is surfaced read-only so the user can see
// what the system is using.
type ServiceDNS struct {
	Service string   `json:"service"`
	Servers []string `json:"servers"`
	DHCP    bool     `json:"dhcp"` // true when no manual servers are set (automatic/DHCP)
}

// DNSStatus returns the current DNS configuration for every network service, in
// one call, for a read-only status view.
func (a *App) DNSStatus() ([]ServiceDNS, error) {
	if a.resolver == nil {
		return nil, a.resolverErr
	}
	services, err := a.resolver.ListServices()
	if err != nil {
		return nil, err
	}
	out := make([]ServiceDNS, 0, len(services))
	for _, svc := range services {
		servers, err := a.resolver.CurrentDNS(svc)
		if err != nil {
			return nil, err
		}
		out = append(out, ServiceDNS{Service: svc, Servers: servers, DHCP: len(servers) == 0})
	}
	return out, nil
}

// ListServices returns the available network services/interfaces.
func (a *App) ListServices() ([]string, error) {
	if a.resolver == nil {
		return nil, a.resolverErr
	}
	return a.resolver.ListServices()
}

// CurrentDNS returns the DNS servers currently set for a network service.
func (a *App) CurrentDNS(service string) ([]string, error) {
	if a.resolver == nil {
		return nil, a.resolverErr
	}
	return a.resolver.CurrentDNS(service)
}

// SetDNS applies DNS servers to a network service.
func (a *App) SetDNS(service string, servers []string) error {
	if a.resolver == nil {
		return a.resolverErr
	}
	return a.resolver.Set(service, servers)
}

// ClearDNS reverts a network service to DHCP-provided DNS.
func (a *App) ClearDNS(service string) error {
	if a.resolver == nil {
		return a.resolverErr
	}
	return a.resolver.Clear(service)
}

// FlushDNS flushes the DNS cache.
func (a *App) FlushDNS() error {
	if a.resolver == nil {
		return a.resolverErr
	}
	return a.resolver.Flush()
}
