//go:build darwin

package dns

import (
	"fmt"
	"os/exec"
	"strings"
)

// macOSClient provides DNS management operations for macOS.
type macOSClient struct{}

// NewClient creates a new DNS client for the current platform.
func NewClient() (Client, error) {
	return &macOSClient{}, nil
}

// Name returns the backend name for display purposes.
func (c *macOSClient) Name() string {
	return "macOS networksetup"
}

// ListNetworkServices returns all available network services.
func (c *macOSClient) ListNetworkServices() ([]string, error) {
	cmd := exec.Command("networksetup", "-listallnetworkservices")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list network services: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var services []string

	for _, line := range lines {
		// Skip the first line which is a header about asterisks
		if strings.Contains(line, "asterisk") {
			continue
		}
		// Skip disabled services (marked with asterisk)
		if strings.HasPrefix(line, "*") {
			continue
		}
		if line != "" {
			services = append(services, line)
		}
	}

	return services, nil
}

// GetDNSServers returns the current DNS servers for a network service.
func (c *macOSClient) GetDNSServers(service string) ([]string, error) {
	cmd := exec.Command("networksetup", "-getdnsservers", service)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get DNS servers for %s: %w", service, err)
	}

	text := strings.TrimSpace(string(output))

	// Check if using DHCP (no manual DNS set)
	if strings.Contains(text, "There aren't any DNS Servers set") {
		return nil, nil
	}

	servers := strings.Split(text, "\n")
	var result []string
	for _, s := range servers {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}

	return result, nil
}

// SetDNSServers sets the DNS servers for a network service.
func (c *macOSClient) SetDNSServers(service string, servers []string) error {
	args := []string{"-setdnsservers", service}
	args = append(args, servers...)

	cmd := exec.Command("networksetup", args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set DNS servers: %s: %w", string(output), err)
	}

	return nil
}

// ClearDNSServers clears DNS servers, reverting to DHCP defaults.
func (c *macOSClient) ClearDNSServers(service string) error {
	cmd := exec.Command("networksetup", "-setdnsservers", service, "empty")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clear DNS servers: %s: %w", string(output), err)
	}

	return nil
}

// PrimaryService returns the network service that currently carries the default
// route (the "active" interface) — e.g. "Wi-Fi". It resolves the default
// route's interface, then maps that interface to its service name.
func (c *macOSClient) PrimaryService() (string, error) {
	iface, err := defaultRouteInterface()
	if err != nil {
		return "", err
	}
	return serviceForInterface(iface)
}

// defaultRouteInterface returns the interface name (e.g. "en0") for the default
// route.
func defaultRouteInterface() (string, error) {
	output, err := exec.Command("route", "-n", "get", "default").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get default route: %w", err)
	}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "interface:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "interface:")), nil
		}
	}
	return "", fmt.Errorf("no default route interface found")
}

// serviceForInterface maps an interface name (e.g. "en0") to its network
// service name (e.g. "Wi-Fi") using the service order listing.
func serviceForInterface(iface string) (string, error) {
	output, err := exec.Command("networksetup", "-listnetworkserviceorder").Output()
	if err != nil {
		return "", fmt.Errorf("failed to list network service order: %w", err)
	}

	// The listing alternates a "(N) Service Name" line with a
	// "(Hardware Port: ..., Device: enX)" line. Track the most recent name and
	// return it when its hardware line names our interface.
	var currentName string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.Contains(line, "Hardware Port"):
			if strings.Contains(line, "Device: "+iface+")") {
				return currentName, nil
			}
		case strings.HasPrefix(line, "("):
			if idx := strings.Index(line, ") "); idx != -1 {
				currentName = strings.TrimSpace(line[idx+2:])
			}
		}
	}
	return "", fmt.Errorf("no network service found for interface %s", iface)
}

// FlushCache flushes the DNS cache.
func (c *macOSClient) FlushCache() error {
	cmd := exec.Command("dscacheutil", "-flushcache")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to flush DNS cache: %s: %w", string(output), err)
	}

	// Also kill mDNSResponder to fully flush on newer macOS versions
	cmd = exec.Command("killall", "-HUP", "mDNSResponder")
	// Ignore errors for this command as it may require elevated privileges
	_ = cmd.Run()

	return nil
}
