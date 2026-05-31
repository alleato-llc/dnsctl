package tui

import (
	"strings"
	"testing"

	"github.com/nycjv321/dnsctl/internal/dns"
)

// TestRenderMainView_ShowsCurrentDNS tests that main view displays DNS servers.
func TestRenderMainView_ShowsCurrentDNS(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentDNS = []string{"1.1.1.1", "1.0.0.1"}
	model.currentService = "Wi-Fi"

	output := model.renderMainView()

	if !strings.Contains(output, "1.1.1.1") {
		t.Error("expected output to contain DNS server 1.1.1.1")
	}
	if !strings.Contains(output, "1.0.0.1") {
		t.Error("expected output to contain DNS server 1.0.0.1")
	}
	if !strings.Contains(output, "Wi-Fi") {
		t.Error("expected output to contain service name Wi-Fi")
	}
}

// TestRenderMainView_ShowsDHCP tests that main view shows DHCP when no DNS set.
func TestRenderMainView_ShowsDHCP(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentDNS = nil
	model.currentService = "Wi-Fi"

	output := model.renderMainView()

	if !strings.Contains(output, "DHCP") {
		t.Error("expected output to contain DHCP")
	}
}

// TestRenderMainView_ShowsStatusMessage tests that status messages are displayed.
func TestRenderMainView_ShowsStatusMessage(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.statusMsg = "Applied profile: cloudflare"
	model.statusIsError = false

	output := model.renderMainView()

	if !strings.Contains(output, "Applied profile: cloudflare") {
		t.Error("expected output to contain status message")
	}
}

// TestRenderMainView_ShowsErrorStatus tests that error status is displayed.
func TestRenderMainView_ShowsErrorStatus(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.statusMsg = "Error: permission denied"
	model.statusIsError = true

	output := model.renderMainView()

	if !strings.Contains(output, "Error: permission denied") {
		t.Error("expected output to contain error message")
	}
}

// TestRenderMainView_ShowsTitle tests that main view shows title.
func TestRenderMainView_ShowsTitle(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))

	output := model.renderMainView()

	if !strings.Contains(output, "DNS Profile Switcher") {
		t.Error("expected output to contain title")
	}
}

// TestRenderMainView_ShowsHelp tests that main view shows help text.
func TestRenderMainView_ShowsHelp(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))

	output := model.renderMainView()

	if !strings.Contains(output, "[p]") {
		t.Error("expected output to contain [p] key hint")
	}
	if !strings.Contains(output, "[q]") {
		t.Error("expected output to contain [q] key hint")
	}
	if !strings.Contains(output, "switch profile") {
		t.Error("expected output to contain 'switch profile'")
	}
}

// TestRenderProfilesView_ListsProfiles tests that profiles view lists all profiles.
func TestRenderProfilesView_ListsProfiles(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewProfiles

	output := model.renderProfilesView()

	if !strings.Contains(output, "cloudflare") {
		t.Error("expected output to contain cloudflare profile")
	}
	if !strings.Contains(output, "google") {
		t.Error("expected output to contain google profile")
	}
	if !strings.Contains(output, "dhcp") {
		t.Error("expected output to contain dhcp profile")
	}
}

// TestRenderProfilesView_HighlightsSelected tests that selected profile is highlighted.
func TestRenderProfilesView_HighlightsSelected(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewProfiles
	model.selectedIndex = 0

	output := model.renderProfilesView()

	// The selected item should have a cursor
	if !strings.Contains(output, "> ") {
		t.Error("expected output to contain selection cursor")
	}
}

// TestRenderProfilesView_ShowsDescription tests that selected profile shows description.
func TestRenderProfilesView_ShowsDescription(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewProfiles
	model.selectedIndex = 0 // cloudflare is first alphabetically

	output := model.renderProfilesView()

	if !strings.Contains(output, "Cloudflare DNS") {
		t.Error("expected output to contain profile description")
	}
}

// TestRenderProfilesView_ShowsServers tests that selected profile shows servers.
func TestRenderProfilesView_ShowsServers(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewProfiles
	model.selectedIndex = 0 // cloudflare

	output := model.renderProfilesView()

	if !strings.Contains(output, "1.1.1.1") {
		t.Error("expected output to contain DNS servers")
	}
}

// TestRenderProfilesView_ShowsDHCPForDHCPProfile tests DHCP indicator.
func TestRenderProfilesView_ShowsDHCPForDHCPProfile(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewProfiles
	// Find dhcp profile index
	names := cfg.ProfileNames()
	for i, name := range names {
		if name == "dhcp" {
			model.selectedIndex = i
			break
		}
	}

	output := model.renderProfilesView()

	if !strings.Contains(output, "DHCP") {
		t.Error("expected output to contain DHCP indicator for dhcp profile")
	}
}

// TestRenderProfilesView_ShowsTitle tests that profiles view shows title.
func TestRenderProfilesView_ShowsTitle(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewProfiles

	output := model.renderProfilesView()

	if !strings.Contains(output, "Select DNS Profile") {
		t.Error("expected output to contain profiles view title")
	}
}

// TestRenderServicesView_ListsServices tests that services view lists all services.
func TestRenderServicesView_ListsServices(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewServices
	model.services = []string{"Wi-Fi", "Ethernet", "Thunderbolt Bridge"}

	output := model.renderServicesView()

	if !strings.Contains(output, "Wi-Fi") {
		t.Error("expected output to contain Wi-Fi")
	}
	if !strings.Contains(output, "Ethernet") {
		t.Error("expected output to contain Ethernet")
	}
	if !strings.Contains(output, "Thunderbolt Bridge") {
		t.Error("expected output to contain Thunderbolt Bridge")
	}
}

// TestRenderServicesView_HighlightsSelected tests that selected service is highlighted.
func TestRenderServicesView_HighlightsSelected(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewServices
	model.services = []string{"Wi-Fi", "Ethernet"}
	model.selectedIndex = 1

	output := model.renderServicesView()

	if !strings.Contains(output, "> ") {
		t.Error("expected output to contain selection cursor")
	}
}

// TestRenderServicesView_ShowsCurrentMarker tests current service marker.
func TestRenderServicesView_ShowsCurrentMarker(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewServices
	model.services = []string{"Wi-Fi", "Ethernet"}
	model.currentService = "Wi-Fi"

	output := model.renderServicesView()

	if !strings.Contains(output, "(current)") {
		t.Error("expected output to contain (current) marker")
	}
}

// TestRenderServicesView_ShowsTitle tests that services view shows title.
func TestRenderServicesView_ShowsTitle(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewServices
	model.services = []string{"Wi-Fi"}

	output := model.renderServicesView()

	if !strings.Contains(output, "Select Network Service") {
		t.Error("expected output to contain services view title")
	}
}

// TestGetSelectedProfile_ReturnsCorrectProfile tests profile selection helper.
func TestGetSelectedProfile_ReturnsCorrectProfile(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.selectedIndex = 0 // cloudflare (first alphabetically)

	name, profile, ok := model.getSelectedProfile()

	if !ok {
		t.Fatal("expected ok to be true")
	}
	if name != "cloudflare" {
		t.Errorf("expected cloudflare, got %s", name)
	}
	if profile.Servers[0] != "1.1.1.1" {
		t.Errorf("expected 1.1.1.1, got %s", profile.Servers[0])
	}
}

// TestGetSelectedProfile_InvalidIndex tests invalid index handling.
func TestGetSelectedProfile_InvalidIndex(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.selectedIndex = 100

	_, _, ok := model.getSelectedProfile()

	if ok {
		t.Error("expected ok to be false for invalid index")
	}
}

// TestGetSelectedProfile_NegativeIndex tests negative index handling.
func TestGetSelectedProfile_NegativeIndex(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.selectedIndex = -1

	_, _, ok := model.getSelectedProfile()

	if ok {
		t.Error("expected ok to be false for negative index")
	}
}

// TestGetSelectedService_ReturnsCorrectService tests service selection helper.
func TestGetSelectedService_ReturnsCorrectService(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.services = []string{"Wi-Fi", "Ethernet"}
	model.selectedIndex = 1

	service, ok := model.getSelectedService()

	if !ok {
		t.Fatal("expected ok to be true")
	}
	if service != "Ethernet" {
		t.Errorf("expected Ethernet, got %s", service)
	}
}

// TestGetSelectedService_InvalidIndex tests invalid index handling.
func TestGetSelectedService_InvalidIndex(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.services = []string{"Wi-Fi"}
	model.selectedIndex = 100

	_, ok := model.getSelectedService()

	if ok {
		t.Error("expected ok to be false for invalid index")
	}
}

// TestGetSelectedService_NegativeIndex tests negative index handling.
func TestGetSelectedService_NegativeIndex(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.services = []string{"Wi-Fi"}
	model.selectedIndex = -1

	_, ok := model.getSelectedService()

	if ok {
		t.Error("expected ok to be false for negative index")
	}
}

// TestView_ReturnsMainViewByDefault tests View() returns main view.
func TestView_ReturnsMainViewByDefault(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewMain

	output := model.View()

	if !strings.Contains(output, "DNS Profile Switcher") {
		t.Error("expected main view output")
	}
}

// TestView_ReturnsProfilesView tests View() returns profiles view.
func TestView_ReturnsProfilesView(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewProfiles

	output := model.View()

	if !strings.Contains(output, "Select DNS Profile") {
		t.Error("expected profiles view output")
	}
}

// TestView_ReturnsServicesView tests View() returns services view.
func TestView_ReturnsServicesView(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.currentView = ViewServices
	model.services = []string{"Wi-Fi"}

	output := model.View()

	if !strings.Contains(output, "Select Network Service") {
		t.Error("expected services view output")
	}
}

// TestRenderListHelp tests the list help rendering.
func TestRenderListHelp(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))

	output := model.renderListHelp()

	if !strings.Contains(output, "navigate") {
		t.Error("expected help to contain 'navigate'")
	}
	if !strings.Contains(output, "select") {
		t.Error("expected help to contain 'select'")
	}
	if !strings.Contains(output, "back") {
		t.Error("expected help to contain 'back'")
	}
	if !strings.Contains(output, "quit") {
		t.Error("expected help to contain 'quit'")
	}
}

// TestRenderMainHelp tests the main help rendering.
func TestRenderMainHelp(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))

	output := model.renderMainHelp()

	if !strings.Contains(output, "switch profile") {
		t.Error("expected help to contain 'switch profile'")
	}
	if !strings.Contains(output, "clear DNS") {
		t.Error("expected help to contain 'clear DNS'")
	}
	if !strings.Contains(output, "change service") {
		t.Error("expected help to contain 'change service'")
	}
	if !strings.Contains(output, "refresh") {
		t.Error("expected help to contain 'refresh'")
	}
	if !strings.Contains(output, "quit") {
		t.Error("expected help to contain 'quit'")
	}
}
