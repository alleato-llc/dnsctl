package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nycjv321/dnsctl/internal/config"
	"github.com/nycjv321/dnsctl/internal/dns"
)

// newTestProfileService builds a ProfileService backed by a temp config file
// seeded with the given config, plus a resolver whose writes are recorded.
func newTestProfileService(t *testing.T, cfg *config.Config) (*ProfileService, *recordingRunner, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if cfg != nil {
		if err := cfg.Save(path); err != nil {
			t.Fatalf("seed config: %v", err)
		}
	}
	runner := &recordingRunner{}
	resolver := NewResolverService(dns.NewMockClient(), runner)
	return NewProfileService(path, resolver), runner, path
}

func seedConfig() *config.Config {
	return &config.Config{
		Version:        1,
		DefaultService: "Wi-Fi",
		Profiles: map[string]config.Profile{
			"cloudflare": {Description: "Cloudflare", Servers: []string{"1.1.1.1", "1.0.0.1"}},
			"travel":     {Description: "DHCP", DHCP: true},
		},
		Settings: config.Settings{FlushCache: true},
	}
}

func TestProfileService_List(t *testing.T) {
	svc, _, _ := newTestProfileService(t, seedConfig())
	list, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(list))
	}
	// Sorted by name: cloudflare before travel.
	if list[0].Name != "cloudflare" || list[1].Name != "travel" {
		t.Errorf("unexpected order: %q, %q", list[0].Name, list[1].Name)
	}
	if !list[1].DHCP {
		t.Error("travel profile should report DHCP")
	}
}

func TestProfileService_ListEmptyIsNonNil(t *testing.T) {
	// No config file -> defaults; but force an empty profile set.
	svc, _, path := newTestProfileService(t, &config.Config{Version: 1})
	_ = path
	list, err := svc.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if list == nil {
		t.Error("empty list should be non-nil (marshals to [] not null)")
	}
}

func TestProfileService_ApplyServersSetsDNS(t *testing.T) {
	svc, runner, _ := newTestProfileService(t, seedConfig())
	if err := svc.Apply("cloudflare", "Ethernet"); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(runner.setCalls) != 1 {
		t.Fatalf("expected 1 SetDNS call, got %d", len(runner.setCalls))
	}
	if runner.setCalls[0].Service != "Ethernet" {
		t.Errorf("applied to %q, want Ethernet", runner.setCalls[0].Service)
	}
	if len(runner.setCalls[0].Servers) != 2 {
		t.Errorf("expected 2 servers, got %v", runner.setCalls[0].Servers)
	}
	// flush_cache is set in seed config.
	if runner.flushed != 1 {
		t.Errorf("expected a cache flush, got %d", runner.flushed)
	}
}

func TestProfileService_ApplyDHCPClearsDNS(t *testing.T) {
	svc, runner, _ := newTestProfileService(t, seedConfig())
	if err := svc.Apply("travel", "Wi-Fi"); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(runner.clearCalls) != 1 || runner.clearCalls[0] != "Wi-Fi" {
		t.Errorf("expected Clear on Wi-Fi, got %v", runner.clearCalls)
	}
	if len(runner.setCalls) != 0 {
		t.Errorf("DHCP profile must not set servers, got %v", runner.setCalls)
	}
}

func TestProfileService_ApplyDefaultsToConfigService(t *testing.T) {
	svc, runner, _ := newTestProfileService(t, seedConfig())
	// Empty service falls back to config DefaultService ("Wi-Fi").
	if err := svc.Apply("cloudflare", ""); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(runner.setCalls) != 1 || runner.setCalls[0].Service != "Wi-Fi" {
		t.Errorf("expected fallback to Wi-Fi, got %v", runner.setCalls)
	}
}

func TestProfileService_ApplyUnknown(t *testing.T) {
	svc, _, _ := newTestProfileService(t, seedConfig())
	err := svc.Apply("nope", "Wi-Fi")
	if !errors.Is(err, ErrNoProfile) {
		t.Errorf("expected ErrNoProfile, got %v", err)
	}
}

func TestProfileService_SaveAndDeleteRoundTrip(t *testing.T) {
	svc, _, path := newTestProfileService(t, seedConfig())

	if err := svc.Save(Profile{Name: "quad9", Description: "Quad9", Servers: []string{"9.9.9.9"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := svc.Get("quad9")
	if err != nil {
		t.Fatalf("Get after Save: %v", err)
	}
	if got.Servers[0] != "9.9.9.9" {
		t.Errorf("unexpected servers: %v", got.Servers)
	}
	// Persisted to disk.
	if _, statErr := os.Stat(path); statErr != nil {
		t.Errorf("config not written: %v", statErr)
	}

	if err := svc.Delete("quad9"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := svc.Get("quad9"); !errors.Is(err, ErrNoProfile) {
		t.Errorf("expected profile gone, got %v", err)
	}
}

func TestProfileService_SaveRequiresName(t *testing.T) {
	svc, _, _ := newTestProfileService(t, seedConfig())
	if err := svc.Save(Profile{Servers: []string{"1.1.1.1"}}); err == nil {
		t.Error("expected error when name is empty")
	}
}

func TestProfileService_DeleteUnknown(t *testing.T) {
	svc, _, _ := newTestProfileService(t, seedConfig())
	if err := svc.Delete("nope"); !errors.Is(err, ErrNoProfile) {
		t.Errorf("expected ErrNoProfile, got %v", err)
	}
}

func TestProfileService_ApplyWithoutResolver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := seedConfig().Save(path); err != nil {
		t.Fatalf("seed: %v", err)
	}
	svc := NewProfileService(path, nil)
	if err := svc.Apply("cloudflare", "Wi-Fi"); err == nil {
		t.Error("expected error applying without a resolver")
	}
	// Read/edit still work without a resolver.
	if _, err := svc.List(); err != nil {
		t.Errorf("List without resolver: %v", err)
	}
}
