package service

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/nycjv321/dnsctl/internal/hosts"
)

// recordingRunner is a test PrivilegedRunner that performs the hosts write to a
// real (temp) file and records DNS-side calls without needing root or a DNS
// backend.
type recordingRunner struct {
	flushed   int
	saveErr   error
	lastWrite []byte
}

func (r *recordingRunner) SetDNS(string, []string) error { return nil }
func (r *recordingRunner) ClearDNS(string) error         { return nil }
func (r *recordingRunner) FlushDNS() error               { r.flushed++; return nil }
func (r *recordingRunner) SaveHosts(path string, content []byte) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.lastWrite = content
	return os.WriteFile(path, content, 0644)
}

func newTestService(t *testing.T) (*HostsService, *recordingRunner, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hosts")
	runner := &recordingRunner{}
	return NewHostsService(path, runner), runner, path
}

func TestHostsService_AddThenGetAndList(t *testing.T) {
	svc, _, _ := newTestService(t)

	if _, err := svc.Add(hosts.Entry{IP: "1.2.3.4", Hostname: "a.local"}, ApplyOptions{}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, ok, err := svc.Get("a.local")
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if got.IP != "1.2.3.4" {
		t.Errorf("got IP %q", got.IP)
	}
	list, err := svc.List()
	if err != nil || len(list) != 1 {
		t.Fatalf("List len=%d err=%v", len(list), err)
	}
}

func TestHostsService_AddExisting(t *testing.T) {
	svc, _, _ := newTestService(t)
	entry := hosts.Entry{IP: "1.2.3.4", Hostname: "a.local"}
	if _, err := svc.Add(entry, ApplyOptions{}); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	_, err := svc.Add(entry, ApplyOptions{})
	if !errors.Is(err, ErrExists) {
		t.Errorf("expected ErrExists, got %v", err)
	}
}

func TestHostsService_SetIsIdempotent(t *testing.T) {
	svc, _, _ := newTestService(t)
	if _, err := svc.Set(hosts.Entry{IP: "1.2.3.4", Hostname: "a.local"}, ApplyOptions{}); err != nil {
		t.Fatalf("first Set: %v", err)
	}
	if _, err := svc.Set(hosts.Entry{IP: "5.6.7.8", Hostname: "a.local"}, ApplyOptions{}); err != nil {
		t.Fatalf("second Set: %v", err)
	}
	got, _, _ := svc.Get("a.local")
	if got.IP != "5.6.7.8" {
		t.Errorf("expected updated IP, got %q", got.IP)
	}
}

func TestHostsService_RemoveMissing(t *testing.T) {
	svc, _, _ := newTestService(t)
	_, err := svc.Remove("missing.local", ApplyOptions{})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestHostsService_InvalidEntryRejected(t *testing.T) {
	svc, _, path := newTestService(t)
	_, err := svc.Add(hosts.Entry{IP: "999.1.1.1", Hostname: "a.local"}, ApplyOptions{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("invalid entry should not have written the file")
	}
}

func TestHostsService_DryRunDoesNotWrite(t *testing.T) {
	svc, runner, path := newTestService(t)
	content, err := svc.Add(hosts.Entry{IP: "1.2.3.4", Hostname: "a.local"}, ApplyOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Add dry-run: %v", err)
	}
	if len(content) == 0 {
		t.Error("dry-run should still return rendered content")
	}
	if runner.lastWrite != nil {
		t.Error("dry-run must not call SaveHosts")
	}
	if _, statErr := os.Stat(path); statErr == nil {
		t.Error("dry-run must not create the file")
	}
}

func TestHostsService_FlushAfterWrite(t *testing.T) {
	svc, runner, _ := newTestService(t)
	if _, err := svc.Set(hosts.Entry{IP: "1.2.3.4", Hostname: "a.local"}, ApplyOptions{Flush: true}); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if runner.flushed != 1 {
		t.Errorf("expected 1 flush, got %d", runner.flushed)
	}
}
