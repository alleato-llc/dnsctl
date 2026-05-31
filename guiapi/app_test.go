package guiapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nycjv321/dnsctl/internal/dns"
	"github.com/nycjv321/dnsctl/internal/hosts"
	"github.com/nycjv321/dnsctl/internal/service"
)

// fakeRunner is a PrivilegedRunner that writes hosts to a real (temp) file and
// records resolver calls, so the App can be tested without root or a helper.
type fakeRunner struct {
	setCalls   []string
	clearCalls []string
	flushes    int
}

func (r *fakeRunner) SetDNS(service string, servers []string) error {
	r.setCalls = append(r.setCalls, service)
	return nil
}
func (r *fakeRunner) ClearDNS(service string) error {
	r.clearCalls = append(r.clearCalls, service)
	return nil
}
func (r *fakeRunner) FlushDNS() error { r.flushes++; return nil }
func (r *fakeRunner) SaveHosts(path string, content []byte) error {
	return os.WriteFile(path, content, 0644)
}

func newTestApp(t *testing.T) (*App, *fakeRunner) {
	t.Helper()
	runner := &fakeRunner{}
	hostsPath := filepath.Join(t.TempDir(), "hosts")
	mockDNS := dns.NewMockClient()
	app := New(
		service.NewHostsService(hostsPath, runner),
		service.NewResolverService(mockDNS, runner),
	)
	return app, runner
}

func TestApp_HostsLifecycle(t *testing.T) {
	app, _ := newTestApp(t)

	if err := app.AddHost(hosts.Entry{IP: "127.0.0.1", Hostname: "app.local"}); err != nil {
		t.Fatalf("AddHost: %v", err)
	}
	list, err := app.ListHosts()
	if err != nil || len(list) != 1 {
		t.Fatalf("ListHosts len=%d err=%v", len(list), err)
	}

	if err := app.SetHost(hosts.Entry{IP: "10.0.0.1", Hostname: "app.local"}); err != nil {
		t.Fatalf("SetHost: %v", err)
	}
	list, _ = app.ListHosts()
	if list[0].IP != "10.0.0.1" {
		t.Errorf("expected updated IP, got %s", list[0].IP)
	}

	if err := app.RemoveHost("app.local"); err != nil {
		t.Fatalf("RemoveHost: %v", err)
	}
	list, _ = app.ListHosts()
	if len(list) != 0 {
		t.Errorf("expected empty after remove, got %d", len(list))
	}
}

func TestApp_AddHostValidationError(t *testing.T) {
	app, _ := newTestApp(t)
	if err := app.AddHost(hosts.Entry{IP: "999.1.1.1", Hostname: "bad.local"}); err == nil {
		t.Error("expected validation error for invalid IP")
	}
}

func TestApp_DNSStatus(t *testing.T) {
	runner := &fakeRunner{}
	mockDNS := dns.NewMockClient() // services: Wi-Fi, Ethernet
	mockDNS.DNSServers["Wi-Fi"] = []string{"1.1.1.1", "1.0.0.1"}
	// Ethernet left empty -> should report DHCP.
	app := New(
		service.NewHostsService(filepath.Join(t.TempDir(), "hosts"), runner),
		service.NewResolverService(mockDNS, runner),
	)

	status, err := app.DNSStatus()
	if err != nil {
		t.Fatalf("DNSStatus: %v", err)
	}
	if len(status) != 2 {
		t.Fatalf("expected 2 services, got %d", len(status))
	}

	byName := map[string]ServiceDNS{}
	for _, s := range status {
		byName[s.Service] = s
	}
	if wifi := byName["Wi-Fi"]; wifi.DHCP || len(wifi.Servers) != 2 {
		t.Errorf("Wi-Fi: expected 2 manual servers, got %+v", wifi)
	}
	if eth := byName["Ethernet"]; !eth.DHCP || len(eth.Servers) != 0 {
		t.Errorf("Ethernet: expected DHCP/no servers, got %+v", eth)
	}
}

func TestApp_ResolverRoutesThroughRunner(t *testing.T) {
	app, runner := newTestApp(t)

	services, err := app.ListServices()
	if err != nil || len(services) == 0 {
		t.Fatalf("ListServices err=%v len=%d", err, len(services))
	}
	if err := app.SetDNS("Wi-Fi", []string{"1.1.1.1"}); err != nil {
		t.Fatalf("SetDNS: %v", err)
	}
	if err := app.ClearDNS("Wi-Fi"); err != nil {
		t.Fatalf("ClearDNS: %v", err)
	}
	if len(runner.setCalls) != 1 || len(runner.clearCalls) != 1 {
		t.Errorf("expected runner to record 1 set + 1 clear, got %d/%d", len(runner.setCalls), len(runner.clearCalls))
	}
}

func TestApp_ResolverUnavailable(t *testing.T) {
	// hosts-only App: resolver nil, resolverErr set.
	app := New(service.NewHostsService(filepath.Join(t.TempDir(), "hosts"), &fakeRunner{}), nil)
	app.resolverErr = dns.ErrNoDNSBackend
	if _, err := app.ListServices(); err == nil {
		t.Error("expected error when resolver unavailable")
	}
}
