package dns

// SetDNSCall records a call to SetDNSServers.
type SetDNSCall struct {
	Service string
	Servers []string
}

// MockClient is a mock implementation of the DNS Client interface for testing.
type MockClient struct {
	// Configurable responses
	Services   []string
	DNSServers map[string][]string
	Primary    string // service reported as the active/default one

	// Error injection
	ListError  error
	GetError   error
	SetError   error
	ClearError error
	FlushError error

	// Call recording
	SetCalls   []SetDNSCall
	ClearCalls []string
	FlushCalls int
}

// NewMockClient creates a new mock DNS client with sensible defaults.
func NewMockClient() *MockClient {
	return &MockClient{
		Services:   []string{"Wi-Fi", "Ethernet"},
		DNSServers: make(map[string][]string),
	}
}

// ListNetworkServices returns the configured services list.
func (m *MockClient) ListNetworkServices() ([]string, error) {
	if m.ListError != nil {
		return nil, m.ListError
	}
	return m.Services, nil
}

// GetDNSServers returns the DNS servers for the specified service.
func (m *MockClient) GetDNSServers(service string) ([]string, error) {
	if m.GetError != nil {
		return nil, m.GetError
	}
	if m.DNSServers == nil {
		return nil, nil
	}
	return m.DNSServers[service], nil
}

// SetDNSServers records the call and optionally returns an error.
func (m *MockClient) SetDNSServers(service string, servers []string) error {
	m.SetCalls = append(m.SetCalls, SetDNSCall{
		Service: service,
		Servers: servers,
	})
	if m.SetError != nil {
		return m.SetError
	}
	if m.DNSServers == nil {
		m.DNSServers = make(map[string][]string)
	}
	m.DNSServers[service] = servers
	return nil
}

// ClearDNSServers records the call and optionally returns an error.
func (m *MockClient) ClearDNSServers(service string) error {
	m.ClearCalls = append(m.ClearCalls, service)
	if m.ClearError != nil {
		return m.ClearError
	}
	if m.DNSServers != nil {
		delete(m.DNSServers, service)
	}
	return nil
}

// FlushCache records the call and optionally returns an error.
func (m *MockClient) FlushCache() error {
	m.FlushCalls++
	if m.FlushError != nil {
		return m.FlushError
	}
	return nil
}

// Name returns the mock client name.
func (m *MockClient) Name() string {
	return "mock"
}

// PrimaryService returns the configured active/default service. It implements
// the optional primary-service capability used by ResolverService.
func (m *MockClient) PrimaryService() (string, error) {
	return m.Primary, nil
}
