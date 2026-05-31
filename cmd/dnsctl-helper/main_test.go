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
// are restricted to hostsPath, authorizing the given UIDs (in addition to
// root), and returns a HelperClient pointed at it.
func startHelper(t *testing.T, hostsPath string, allowedUIDs map[uint32]bool) *service.HelperClient {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "helper.sock")

	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	h := &handler{runner: service.NewDirectRunner(), allowedHostsPath: hostsPath, allowedUIDs: allowedUIDs}
	go func() { _ = serve(ln, h) }()
	t.Cleanup(func() { ln.Close() })

	return &service.HelperClient{SocketPath: sock}
}

// selfUID is the set authorizing the current test process (the peer UID seen by
// the helper on a same-process connection).
func selfUID() map[uint32]bool {
	return map[uint32]bool{uint32(os.Getuid()): true}
}

func TestHelper_SaveHosts_RoundTrip(t *testing.T) {
	hostsPath := filepath.Join(t.TempDir(), "hosts")
	client := startHelper(t, hostsPath, selfUID())

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
	client := startHelper(t, allowed, selfUID())

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
	client := startHelper(t, hostsPath, selfUID())

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
	client := startHelper(t, filepath.Join(t.TempDir(), "hosts"), selfUID())
	// SetDNS with an invalid IP exercises the validation path without needing root.
	if err := client.SetDNS("Wi-Fi", []string{"not-an-ip"}); err == nil {
		t.Fatal("expected invalid DNS server to be rejected")
	}
}

func TestHelper_UnauthorizedUID(t *testing.T) {
	hostsPath := filepath.Join(t.TempDir(), "hosts")
	// Authorize only a UID that is not us (and not root), so our connection is
	// rejected by the authorization gate.
	notUs := uint32(os.Getuid()) + 99999
	client := startHelper(t, hostsPath, map[uint32]bool{notUs: true})

	err := client.SaveHosts(hostsPath, []byte("# BEGIN dnsctl\n127.0.0.1\tx.local\n# END dnsctl\n"))
	if err == nil {
		t.Fatal("expected unauthorized UID to be rejected")
	}
	if !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("expected authorization error, got: %v", err)
	}
	if _, statErr := os.Stat(hostsPath); statErr == nil {
		t.Error("unauthorized request must not have written the file")
	}
}
