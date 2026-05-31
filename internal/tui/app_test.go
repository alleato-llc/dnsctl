package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nycjv321/dnsctl/internal/config"
	"github.com/nycjv321/dnsctl/internal/dns"
	"github.com/nycjv321/dnsctl/internal/service"
)

// newResolver wraps a (mock) DNS client in a ResolverService whose privileged
// writes are routed back to the same client, so tests can assert on the mock's
// recorded calls.
func newResolver(client dns.Client) *service.ResolverService {
	return service.NewResolverService(client, service.NewDirectRunnerWithClient(client))
}

// testConfig returns a test configuration with known profiles.
func testConfig() *config.Config {
	return &config.Config{
		Version:        1,
		DefaultService: "Wi-Fi",
		Profiles: map[string]config.Profile{
			"cloudflare": {
				Description: "Cloudflare DNS",
				Servers:     []string{"1.1.1.1", "1.0.0.1"},
			},
			"google": {
				Description: "Google Public DNS",
				Servers:     []string{"8.8.8.8", "8.8.4.4"},
			},
			"dhcp": {
				Description: "Use DHCP",
				DHCP:        true,
			},
		},
		Settings: config.Settings{
			FlushCache: true,
		},
	}
}

// testModel creates a test model with a mock DNS client.
func testModel() (Model, *dns.MockClient) {
	mock := dns.NewMockClient()
	mock.DNSServers["Wi-Fi"] = []string{"8.8.8.8", "8.8.4.4"}
	cfg := testConfig()
	model := NewModel(cfg, newResolver(mock))
	model.services = mock.Services
	model.currentDNS = mock.DNSServers["Wi-Fi"]
	return model, mock
}

// TestNewModel_InitializesCorrectly tests that NewModel sets up the model properly.
func TestNewModel_InitializesCorrectly(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()

	model := NewModel(cfg, newResolver(mock))

	if model.config != cfg {
		t.Error("config not set correctly")
	}
	if model.resolver == nil {
		t.Error("resolver not set correctly")
	}
	if model.currentView != ViewMain {
		t.Errorf("expected currentView to be ViewMain, got %v", model.currentView)
	}
	if model.currentService != "Wi-Fi" {
		t.Errorf("expected currentService to be Wi-Fi, got %s", model.currentService)
	}
	if model.selectedIndex != 0 {
		t.Errorf("expected selectedIndex to be 0, got %d", model.selectedIndex)
	}
}

// TestUpdate_StatusMsg_Success tests handling of successful status messages.
func TestUpdate_StatusMsg_Success(t *testing.T) {
	model, _ := testModel()

	msg := statusMsg{
		services:   []string{"Wi-Fi", "Ethernet", "Thunderbolt"},
		dnsServers: []string{"1.1.1.1", "1.0.0.1"},
	}

	newModel, cmd := model.Update(msg)
	m := newModel.(Model)

	if cmd != nil {
		t.Error("expected no command")
	}
	if len(m.services) != 3 {
		t.Errorf("expected 3 services, got %d", len(m.services))
	}
	if len(m.currentDNS) != 2 {
		t.Errorf("expected 2 DNS servers, got %d", len(m.currentDNS))
	}
	if m.currentDNS[0] != "1.1.1.1" {
		t.Errorf("expected first DNS server to be 1.1.1.1, got %s", m.currentDNS[0])
	}
}

// TestUpdate_StatusMsg_Error tests handling of error status messages.
func TestUpdate_StatusMsg_Error(t *testing.T) {
	model, _ := testModel()

	msg := statusMsg{
		err: errors.New("network error"),
	}

	newModel, cmd := model.Update(msg)
	m := newModel.(Model)

	if cmd != nil {
		t.Error("expected no command")
	}
	if m.statusMsg != "Error: network error" {
		t.Errorf("expected error message, got %s", m.statusMsg)
	}
	if !m.statusIsError {
		t.Error("expected statusIsError to be true")
	}
}

// TestUpdate_DNSChangedMsg_Success tests successful DNS change messages.
func TestUpdate_DNSChangedMsg_Success(t *testing.T) {
	model, _ := testModel()

	msg := dnsChangedMsg{
		success: true,
		message: "Applied profile: cloudflare",
	}

	newModel, cmd := model.Update(msg)
	m := newModel.(Model)

	if cmd == nil {
		t.Error("expected command for refresh")
	}
	if m.statusMsg != "Applied profile: cloudflare" {
		t.Errorf("expected status message, got %s", m.statusMsg)
	}
	if m.statusIsError {
		t.Error("expected statusIsError to be false")
	}
}

// TestUpdate_DNSChangedMsg_Error tests failed DNS change messages.
func TestUpdate_DNSChangedMsg_Error(t *testing.T) {
	model, _ := testModel()

	msg := dnsChangedMsg{
		success: false,
		message: "Failed to apply profile: permission denied",
	}

	newModel, cmd := model.Update(msg)
	m := newModel.(Model)

	if cmd == nil {
		t.Error("expected command for refresh")
	}
	if m.statusMsg != "Failed to apply profile: permission denied" {
		t.Errorf("expected error message, got %s", m.statusMsg)
	}
	if !m.statusIsError {
		t.Error("expected statusIsError to be true")
	}
}

// TestUpdate_ClearStatusMsg tests clearing the status message.
func TestUpdate_ClearStatusMsg(t *testing.T) {
	model, _ := testModel()
	model.statusMsg = "Some message"
	model.statusIsError = true

	newModel, cmd := model.Update(clearStatusMsg{})
	m := newModel.(Model)

	if cmd != nil {
		t.Error("expected no command")
	}
	if m.statusMsg != "" {
		t.Errorf("expected empty status message, got %s", m.statusMsg)
	}
	if m.statusIsError {
		t.Error("expected statusIsError to be false")
	}
}

// TestUpdate_WindowSizeMsg tests window size handling.
func TestUpdate_WindowSizeMsg(t *testing.T) {
	model, _ := testModel()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}

	newModel, cmd := model.Update(msg)
	m := newModel.(Model)

	if cmd != nil {
		t.Error("expected no command")
	}
	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
}

// TestMainView_Quit tests quitting from the main view.
func TestMainView_Quit(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewMain

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

	_, cmd := model.Update(msg)

	// The quit command should return a quit message
	if cmd == nil {
		t.Error("expected quit command")
	}
}

// TestMainView_SwitchToProfiles tests switching to profiles view.
func TestMainView_SwitchToProfiles(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewMain
	model.selectedIndex = 5

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}

	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.currentView != ViewProfiles {
		t.Errorf("expected ViewProfiles, got %v", m.currentView)
	}
	if m.selectedIndex != 0 {
		t.Errorf("expected selectedIndex reset to 0, got %d", m.selectedIndex)
	}
	if m.statusMsg != "" {
		t.Errorf("expected empty status message, got %s", m.statusMsg)
	}
}

// TestMainView_SwitchToServices tests switching to services view.
func TestMainView_SwitchToServices(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewMain
	model.currentService = "Ethernet"

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}

	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.currentView != ViewServices {
		t.Errorf("expected ViewServices, got %v", m.currentView)
	}
	// Should find and select current service index
	if m.selectedIndex != 1 {
		t.Errorf("expected selectedIndex to be 1 (Ethernet), got %d", m.selectedIndex)
	}
}

// TestMainView_ClearDNS tests the clear DNS command from main view.
func TestMainView_ClearDNS(t *testing.T) {
	model, mock := testModel()
	model.currentView = ViewMain

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}

	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("expected clear DNS command")
	}

	// Execute the command
	result := cmd()
	dnsMsg, ok := result.(dnsChangedMsg)
	if !ok {
		t.Error("expected dnsChangedMsg")
	}
	if !dnsMsg.success {
		t.Errorf("expected success, got error: %s", dnsMsg.message)
	}
	if len(mock.ClearCalls) != 1 {
		t.Errorf("expected 1 clear call, got %d", len(mock.ClearCalls))
	}
	if mock.ClearCalls[0] != "Wi-Fi" {
		t.Errorf("expected clear call for Wi-Fi, got %s", mock.ClearCalls[0])
	}
}

// TestMainView_Refresh tests the refresh command.
func TestMainView_Refresh(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewMain

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}

	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("expected refresh command")
	}
}

// TestProfilesView_NavigateDown tests navigating down in profiles view.
func TestProfilesView_NavigateDown(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewProfiles
	model.selectedIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}

	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.selectedIndex != 1 {
		t.Errorf("expected selectedIndex 1, got %d", m.selectedIndex)
	}
}

// TestProfilesView_NavigateUp tests navigating up in profiles view.
func TestProfilesView_NavigateUp(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewProfiles
	model.selectedIndex = 2

	msg := tea.KeyMsg{Type: tea.KeyUp}

	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.selectedIndex != 1 {
		t.Errorf("expected selectedIndex 1, got %d", m.selectedIndex)
	}
}

// TestProfilesView_NavigateUp_AtTop tests that navigation stops at top.
func TestProfilesView_NavigateUp_AtTop(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewProfiles
	model.selectedIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyUp}

	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.selectedIndex != 0 {
		t.Errorf("expected selectedIndex to stay 0, got %d", m.selectedIndex)
	}
}

// TestProfilesView_NavigateDown_AtBottom tests that navigation stops at bottom.
func TestProfilesView_NavigateDown_AtBottom(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewProfiles
	profileCount := len(model.config.ProfileNames())
	model.selectedIndex = profileCount - 1

	msg := tea.KeyMsg{Type: tea.KeyDown}

	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.selectedIndex != profileCount-1 {
		t.Errorf("expected selectedIndex to stay %d, got %d", profileCount-1, m.selectedIndex)
	}
}

// TestProfilesView_SelectProfile tests selecting a profile.
func TestProfilesView_SelectProfile(t *testing.T) {
	model, mock := testModel()
	model.currentView = ViewProfiles
	model.selectedIndex = 0 // cloudflare (alphabetically sorted)

	msg := tea.KeyMsg{Type: tea.KeyEnter}

	_, cmd := model.Update(msg)

	if cmd == nil {
		t.Error("expected command to apply profile")
	}

	// Execute the command
	result := cmd()
	dnsMsg, ok := result.(dnsChangedMsg)
	if !ok {
		t.Error("expected dnsChangedMsg")
	}
	if !dnsMsg.success {
		t.Errorf("expected success, got: %s", dnsMsg.message)
	}
	if len(mock.SetCalls) != 1 {
		t.Errorf("expected 1 set call, got %d", len(mock.SetCalls))
	}
	// cloudflare profile
	if mock.SetCalls[0].Servers[0] != "1.1.1.1" {
		t.Errorf("expected 1.1.1.1, got %s", mock.SetCalls[0].Servers[0])
	}
}

// TestProfilesView_Back tests going back from profiles view.
func TestProfilesView_Back(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewProfiles
	model.statusMsg = "some message"

	msg := tea.KeyMsg{Type: tea.KeyEsc}

	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.currentView != ViewMain {
		t.Errorf("expected ViewMain, got %v", m.currentView)
	}
	if m.statusMsg != "" {
		t.Errorf("expected empty status message, got %s", m.statusMsg)
	}
}

// TestServicesView_NavigateDown tests navigating down in services view.
func TestServicesView_NavigateDown(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewServices
	model.selectedIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}

	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.selectedIndex != 1 {
		t.Errorf("expected selectedIndex 1, got %d", m.selectedIndex)
	}
}

// TestServicesView_NavigateUp tests navigating up in services view.
func TestServicesView_NavigateUp(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewServices
	model.selectedIndex = 1

	msg := tea.KeyMsg{Type: tea.KeyUp}

	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.selectedIndex != 0 {
		t.Errorf("expected selectedIndex 0, got %d", m.selectedIndex)
	}
}

// TestServicesView_SelectService tests selecting a service.
func TestServicesView_SelectService(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewServices
	model.selectedIndex = 1 // Ethernet

	msg := tea.KeyMsg{Type: tea.KeyEnter}

	newModel, cmd := model.Update(msg)
	m := newModel.(Model)

	if m.currentService != "Ethernet" {
		t.Errorf("expected Ethernet, got %s", m.currentService)
	}
	if m.currentView != ViewMain {
		t.Errorf("expected ViewMain, got %v", m.currentView)
	}
	if cmd == nil {
		t.Error("expected refresh command")
	}
}

// TestServicesView_Back tests going back from services view.
func TestServicesView_Back(t *testing.T) {
	model, _ := testModel()
	model.currentView = ViewServices

	msg := tea.KeyMsg{Type: tea.KeyEsc}

	newModel, _ := model.Update(msg)
	m := newModel.(Model)

	if m.currentView != ViewMain {
		t.Errorf("expected ViewMain, got %v", m.currentView)
	}
}

// TestApplyProfile_SetsDNS tests that applying a profile sets DNS.
func TestApplyProfile_SetsDNS(t *testing.T) {
	model, mock := testModel()
	profile := config.Profile{
		Description: "Test",
		Servers:     []string{"9.9.9.9"},
	}

	cmd := model.applyProfile("test", profile)
	result := cmd()

	dnsMsg, ok := result.(dnsChangedMsg)
	if !ok {
		t.Fatal("expected dnsChangedMsg")
	}
	if !dnsMsg.success {
		t.Errorf("expected success, got: %s", dnsMsg.message)
	}
	if len(mock.SetCalls) != 1 {
		t.Fatalf("expected 1 set call, got %d", len(mock.SetCalls))
	}
	if mock.SetCalls[0].Service != "Wi-Fi" {
		t.Errorf("expected Wi-Fi, got %s", mock.SetCalls[0].Service)
	}
	if mock.SetCalls[0].Servers[0] != "9.9.9.9" {
		t.Errorf("expected 9.9.9.9, got %s", mock.SetCalls[0].Servers[0])
	}
}

// TestApplyProfile_DHCP_ClearsDNS tests that DHCP profiles clear DNS.
func TestApplyProfile_DHCP_ClearsDNS(t *testing.T) {
	model, mock := testModel()
	profile := config.Profile{
		Description: "Use DHCP",
		DHCP:        true,
	}

	cmd := model.applyProfile("dhcp", profile)
	result := cmd()

	dnsMsg, ok := result.(dnsChangedMsg)
	if !ok {
		t.Fatal("expected dnsChangedMsg")
	}
	if !dnsMsg.success {
		t.Errorf("expected success, got: %s", dnsMsg.message)
	}
	if len(mock.ClearCalls) != 1 {
		t.Fatalf("expected 1 clear call, got %d", len(mock.ClearCalls))
	}
	if mock.ClearCalls[0] != "Wi-Fi" {
		t.Errorf("expected Wi-Fi, got %s", mock.ClearCalls[0])
	}
	if len(mock.SetCalls) != 0 {
		t.Errorf("expected no set calls, got %d", len(mock.SetCalls))
	}
}

// TestApplyProfile_FlushesCache tests that cache is flushed when configured.
func TestApplyProfile_FlushesCache(t *testing.T) {
	model, mock := testModel()
	profile := config.Profile{
		Description: "Test",
		Servers:     []string{"9.9.9.9"},
	}

	cmd := model.applyProfile("test", profile)
	cmd()

	if mock.FlushCalls != 1 {
		t.Errorf("expected 1 flush call, got %d", mock.FlushCalls)
	}
}

// TestApplyProfile_NoFlushWhenDisabled tests that cache is not flushed when disabled.
func TestApplyProfile_NoFlushWhenDisabled(t *testing.T) {
	mock := dns.NewMockClient()
	cfg := testConfig()
	cfg.Settings.FlushCache = false
	model := NewModel(cfg, newResolver(mock))

	profile := config.Profile{
		Description: "Test",
		Servers:     []string{"9.9.9.9"},
	}

	cmd := model.applyProfile("test", profile)
	cmd()

	if mock.FlushCalls != 0 {
		t.Errorf("expected 0 flush calls, got %d", mock.FlushCalls)
	}
}

// TestApplyProfile_Error tests error handling when applying profile fails.
func TestApplyProfile_Error(t *testing.T) {
	model, mock := testModel()
	mock.SetError = errors.New("permission denied")

	profile := config.Profile{
		Description: "Test",
		Servers:     []string{"9.9.9.9"},
	}

	cmd := model.applyProfile("test", profile)
	result := cmd()

	dnsMsg, ok := result.(dnsChangedMsg)
	if !ok {
		t.Fatal("expected dnsChangedMsg")
	}
	if dnsMsg.success {
		t.Error("expected failure")
	}
	if dnsMsg.message != "Failed to apply profile: permission denied" {
		t.Errorf("unexpected message: %s", dnsMsg.message)
	}
}

// TestClearDNS_Success tests successful DNS clearing.
func TestClearDNS_Success(t *testing.T) {
	model, mock := testModel()

	result := model.clearDNS()

	dnsMsg, ok := result.(dnsChangedMsg)
	if !ok {
		t.Fatal("expected dnsChangedMsg")
	}
	if !dnsMsg.success {
		t.Errorf("expected success, got: %s", dnsMsg.message)
	}
	if dnsMsg.message != "DNS cleared (using DHCP)" {
		t.Errorf("unexpected message: %s", dnsMsg.message)
	}
	if len(mock.ClearCalls) != 1 {
		t.Errorf("expected 1 clear call, got %d", len(mock.ClearCalls))
	}
}

// TestClearDNS_Error tests error handling when clearing DNS fails.
func TestClearDNS_Error(t *testing.T) {
	model, mock := testModel()
	mock.ClearError = errors.New("permission denied")

	result := model.clearDNS()

	dnsMsg, ok := result.(dnsChangedMsg)
	if !ok {
		t.Fatal("expected dnsChangedMsg")
	}
	if dnsMsg.success {
		t.Error("expected failure")
	}
	if dnsMsg.message != "Failed to clear DNS: permission denied" {
		t.Errorf("unexpected message: %s", dnsMsg.message)
	}
}

// TestRefreshStatus_Success tests successful status refresh.
func TestRefreshStatus_Success(t *testing.T) {
	model, mock := testModel()
	mock.DNSServers["Wi-Fi"] = []string{"1.1.1.1"}

	result := model.refreshStatus()

	statusResult, ok := result.(statusMsg)
	if !ok {
		t.Fatal("expected statusMsg")
	}
	if statusResult.err != nil {
		t.Errorf("expected no error, got: %v", statusResult.err)
	}
	if len(statusResult.services) != 2 {
		t.Errorf("expected 2 services, got %d", len(statusResult.services))
	}
	if len(statusResult.dnsServers) != 1 {
		t.Errorf("expected 1 DNS server, got %d", len(statusResult.dnsServers))
	}
}

// TestRefreshStatus_ListError tests error handling when listing services fails.
func TestRefreshStatus_ListError(t *testing.T) {
	model, mock := testModel()
	mock.ListError = errors.New("network unavailable")

	result := model.refreshStatus()

	statusResult, ok := result.(statusMsg)
	if !ok {
		t.Fatal("expected statusMsg")
	}
	if statusResult.err == nil {
		t.Error("expected error")
	}
	if statusResult.err.Error() != "network unavailable" {
		t.Errorf("unexpected error: %v", statusResult.err)
	}
}

// TestRefreshStatus_GetError tests error handling when getting DNS fails.
func TestRefreshStatus_GetError(t *testing.T) {
	model, mock := testModel()
	mock.GetError = errors.New("service not found")

	result := model.refreshStatus()

	statusResult, ok := result.(statusMsg)
	if !ok {
		t.Fatal("expected statusMsg")
	}
	if statusResult.err == nil {
		t.Error("expected error")
	}
	if statusResult.err.Error() != "service not found" {
		t.Errorf("unexpected error: %v", statusResult.err)
	}
}
