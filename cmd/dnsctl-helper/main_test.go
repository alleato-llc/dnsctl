package main

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nycjv321/dnsctl/internal/service"
)

// startHelper launches the serve loop on a temp unix socket whose hosts writes
// are restricted to hostsPath, and returns a HelperClient pointed at it.
func startHelper(t *testing.T, hostsPath string) *service.HelperClient {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "helper.sock")

	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	h := &handler{runner: service.NewDirectRunner(), allowedHostsPath: hostsPath}
	go func() { _ = serve(ln, h) }()
	t.Cleanup(func() { ln.Close() })

	return &service.HelperClient{SocketPath: sock}
}

func TestHelper_SaveHosts_RoundTrip(t *testing.T) {
	hostsPath := filepath.Join(t.TempDir(), "hosts")
	client := startHelper(t, hostsPath)

	content := []byte("# BEGIN dnsctl\n127.0.0.1\tapp.local\n# END dnsctl\n")
	if err := client.SaveHosts(hostsPath, content); err != nil {
		t.Fatalf("SaveHosts over IPC: %v", err)
	}

	got, err := os.ReadFile(hostsPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if !strings.Contains(string(got), "app.local") {
		t.Errorf("file not written through helper:\n%s", got)
	}
}

func TestHelper_SaveHosts_DisallowedPath(t *testing.T) {
	allowed := filepath.Join(t.TempDir(), "hosts")
	client := startHelper(t, allowed)

	err := client.SaveHosts("/etc/passwd", []byte("127.0.0.1 x.local\n"))
	if err == nil {
		t.Fatal("expected helper to refuse a disallowed path")
	}
	if !strings.Contains(err.Error(), "disallowed path") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHelper_SaveHosts_InvalidContentRejected(t *testing.T) {
	hostsPath := filepath.Join(t.TempDir(), "hosts")
	client := startHelper(t, hostsPath)

	// A managed entry with a bad IP must be rejected by the trust boundary.
	bad := []byte("# BEGIN dnsctl\n999.1.1.1\tapp.local\n# END dnsctl\n")
	if err := client.SaveHosts(hostsPath, bad); err == nil {
		t.Fatal("expected helper to reject invalid managed entry")
	}
	if _, err := os.Stat(hostsPath); err == nil {
		t.Error("invalid content must not have been written")
	}
}

func TestHelper_UnknownOp(t *testing.T) {
	client := startHelper(t, filepath.Join(t.TempDir(), "hosts"))
	// SetDNS with an invalid IP exercises the validation path without needing root.
	if err := client.SetDNS("Wi-Fi", []string{"not-an-ip"}); err == nil {
		t.Fatal("expected invalid DNS server to be rejected")
	}
}
