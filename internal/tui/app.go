package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nycjv321/dnsctl/internal/config"
	"github.com/nycjv321/dnsctl/internal/service"
)

// Model represents the application state.
type Model struct {
	config         *config.Config
	resolver       *service.ResolverService
	keys           KeyMap
	currentView    View
	currentService string
	currentDNS     []string
	services       []string
	selectedIndex  int
	statusMsg      string
	statusIsError  bool
	width          int
	height         int
}

// NewModel creates a new TUI model backed by the shared resolver facade.
func NewModel(cfg *config.Config, resolver *service.ResolverService) Model {
	return Model{
		config:         cfg,
		resolver:       resolver,
		keys:           DefaultKeyMap(),
		currentView:    ViewMain,
		currentService: cfg.DefaultService,
		selectedIndex:  0,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return m.refreshStatus
}

// refreshStatus fetches the current DNS status.
func (m Model) refreshStatus() tea.Msg {
	// Get network services
	services, err := m.resolver.ListServices()
	if err != nil {
		return statusMsg{err: err}
	}

	// Get current DNS servers
	dnsServers, err := m.resolver.CurrentDNS(m.currentService)
	if err != nil {
		return statusMsg{err: err}
	}

	return statusMsg{
		services:   services,
		dnsServers: dnsServers,
	}
}

// statusMsg is a message containing the current status.
type statusMsg struct {
	services   []string
	dnsServers []string
	err        error
}

// dnsChangedMsg is sent when DNS has been changed.
type dnsChangedMsg struct {
	success bool
	message string
}

// clearStatusMsg is sent to clear the status message after a delay.
type clearStatusMsg struct{}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case statusMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
			m.statusIsError = true
		} else {
			m.services = msg.services
			m.currentDNS = msg.dnsServers
		}
		return m, nil

	case dnsChangedMsg:
		if msg.success {
			m.statusMsg = msg.message
			m.statusIsError = false
		} else {
			m.statusMsg = msg.message
			m.statusIsError = true
		}
		// Refresh status after DNS change and clear status after 3 seconds
		return m, tea.Batch(
			m.refreshStatus,
			tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return clearStatusMsg{}
			}),
		)

	case clearStatusMsg:
		m.statusMsg = ""
		m.statusIsError = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	}

	return m, nil
}

// handleKeyPress handles key press events.
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.currentView {
	case ViewMain:
		return m.handleMainKeys(msg)
	case ViewProfiles:
		return m.handleProfileKeys(msg)
	case ViewServices:
		return m.handleServiceKeys(msg)
	}
	return m, nil
}

// handleMainKeys handles key presses in the main view.
func (m Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.SwitchProfile):
		m.currentView = ViewProfiles
		m.selectedIndex = 0
		m.statusMsg = ""
		return m, nil

	case key.Matches(msg, m.keys.ClearDNS):
		return m, m.clearDNS

	case key.Matches(msg, m.keys.ChangeService):
		m.currentView = ViewServices
		m.selectedIndex = 0
		// Find current service index
		for i, s := range m.services {
			if s == m.currentService {
				m.selectedIndex = i
				break
			}
		}
		m.statusMsg = ""
		return m, nil

	case key.Matches(msg, m.keys.Refresh):
		return m, m.refreshStatus
	}

	return m, nil
}

// handleProfileKeys handles key presses in the profile selection view.
func (m Model) handleProfileKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	profileCount := len(m.config.ProfileNames())

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Back):
		m.currentView = ViewMain
		m.statusMsg = ""
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.selectedIndex < profileCount-1 {
			m.selectedIndex++
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		name, profile, ok := m.getSelectedProfile()
		if ok {
			return m, m.applyProfile(name, profile)
		}
		return m, nil
	}

	return m, nil
}

// handleServiceKeys handles key presses in the service selection view.
func (m Model) handleServiceKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	serviceCount := len(m.services)

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Back):
		m.currentView = ViewMain
		m.statusMsg = ""
		return m, nil

	case key.Matches(msg, m.keys.Up):
		if m.selectedIndex > 0 {
			m.selectedIndex--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.selectedIndex < serviceCount-1 {
			m.selectedIndex++
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		service, ok := m.getSelectedService()
		if ok {
			m.currentService = service
			m.currentView = ViewMain
			return m, m.refreshStatus
		}
		return m, nil
	}

	return m, nil
}

// applyProfile applies a DNS profile.
func (m Model) applyProfile(name string, profile config.Profile) tea.Cmd {
	return func() tea.Msg {
		var err error

		if profile.IsDHCP() {
			// Clear DNS to use DHCP
			err = m.resolver.Clear(m.currentService)
		} else {
			// Set specific DNS servers
			err = m.resolver.Set(m.currentService, profile.Servers)
		}

		if err != nil {
			return dnsChangedMsg{
				success: false,
				message: fmt.Sprintf("Failed to apply profile: %v", err),
			}
		}

		// Flush cache if configured (best-effort)
		if m.config.Settings.FlushCache {
			_ = m.resolver.Flush()
		}

		return dnsChangedMsg{
			success: true,
			message: fmt.Sprintf("Applied profile: %s", name),
		}
	}
}

// clearDNS clears the DNS servers to use DHCP defaults.
func (m Model) clearDNS() tea.Msg {
	err := m.resolver.Clear(m.currentService)
	if err != nil {
		return dnsChangedMsg{
			success: false,
			message: fmt.Sprintf("Failed to clear DNS: %v", err),
		}
	}

	// Flush cache if configured (best-effort)
	if m.config.Settings.FlushCache {
		_ = m.resolver.Flush()
	}

	return dnsChangedMsg{
		success: true,
		message: "DNS cleared (using DHCP)",
	}
}

// View renders the current view.
func (m Model) View() string {
	switch m.currentView {
	case ViewProfiles:
		return m.renderProfilesView()
	case ViewServices:
		return m.renderServicesView()
	default:
		return m.renderMainView()
	}
}
