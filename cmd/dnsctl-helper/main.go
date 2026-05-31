// Command dnsctl-helper is the privileged root daemon that performs dnsctl's
// root-only operations on behalf of unprivileged clients (the GUI, a non-root
// CLI). Clients talk to it over a unix socket using the internal/ipc protocol.
//
// SECURITY: this process runs as root and is the trust boundary. It
// re-validates and authorizes every request and never trusts its caller:
//   - the connecting peer's UID is checked (LOCAL_PEERCRED); only root and
//     UIDs passed via --allow-uids may drive the helper. The socket itself is
//     world-connectable, so this check — not file permissions — is the gate;
//   - hosts writes are restricted to an allow-listed path and the content is
//     re-parsed and validated;
//   - DNS server values are validated as IPs.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/nycjv321/dnsctl/internal/hosts"
	"github.com/nycjv321/dnsctl/internal/ipc"
	"github.com/nycjv321/dnsctl/internal/service"
)

func main() {
	socket := flag.String("socket", ipc.DefaultSocketPath, "unix socket path to listen on")
	hostsPath := flag.String("hosts-file", hosts.DefaultPath, "the only path hosts writes are permitted to target")
	allowUIDs := flag.String("allow-uids", "", "comma-separated UIDs permitted to drive the helper (root is always allowed)")
	flag.Parse()

	allowed, err := parseUIDs(*allowUIDs)
	if err != nil {
		log.Fatalf("parse --allow-uids: %v", err)
	}

	// Remove any stale socket from a previous run before binding.
	if err := os.Remove(*socket); err != nil && !os.IsNotExist(err) {
		log.Fatalf("remove stale socket %s: %v", *socket, err)
	}
	ln, err := net.Listen("unix", *socket)
	if err != nil {
		log.Fatalf("listen on %s: %v", *socket, err)
	}
	defer ln.Close()
	// The socket must be connectable by the unprivileged users the helper
	// serves; access control is enforced by the peer-UID check in authorize(),
	// not by file permissions. Connecting requires write permission on the
	// socket, so make it world-connectable (0666).
	if err := os.Chmod(*socket, 0666); err != nil {
		log.Fatalf("chmod socket: %v", err)
	}

	h := &handler{runner: service.NewDirectRunner(), allowedHostsPath: *hostsPath, allowedUIDs: allowed}
	log.Printf("dnsctl-helper listening on %s (hosts writes limited to %s, allowed UIDs: %v)", *socket, *hostsPath, *allowUIDs)
	if err := serve(ln, h); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// handler dispatches validated requests to the privileged runner.
type handler struct {
	runner           service.PrivilegedRunner
	allowedHostsPath string
	allowedUIDs      map[uint32]bool // root (0) is always allowed
}

// serve accepts connections until the listener is closed, handling each in its
// own goroutine.
func serve(ln net.Listener, h *handler) error {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go h.handle(conn)
	}
}

// handle authorizes the peer, then reads one request, dispatches it, and writes
// one response.
func (h *handler) handle(conn net.Conn) {
	defer conn.Close()

	if err := h.authorize(conn); err != nil {
		_ = ipc.WriteResponse(conn, ipc.Response{Error: err.Error()})
		return
	}

	var req ipc.Request
	if err := ipc.ReadRequest(conn, &req); err != nil {
		_ = ipc.WriteResponse(conn, ipc.Response{Error: fmt.Sprintf("bad request: %v", err)})
		return
	}

	resp := ipc.Response{}
	if err := h.dispatch(req); err != nil {
		resp.Error = err.Error()
	}
	_ = ipc.WriteResponse(conn, resp)
}

// authorize rejects connections from UIDs that are neither root nor on the
// allow list. This is the helper's access-control gate.
func (h *handler) authorize(conn net.Conn) error {
	uid, err := peerUID(conn)
	if err != nil {
		return fmt.Errorf("peer authentication failed: %w", err)
	}
	if uid == 0 || h.allowedUIDs[uid] {
		return nil
	}
	return fmt.Errorf("unauthorized: uid %d is not permitted", uid)
}

// dispatch validates and executes a single request. This is the enforcement
// point for the trust boundary.
func (h *handler) dispatch(req ipc.Request) error {
	switch req.Op {
	case ipc.OpPing:
		// Reaching here means the peer was authorized; nothing else to do.
		return nil

	case ipc.OpSetDNS:
		if err := validateIPs(req.Servers); err != nil {
			return err
		}
		return h.runner.SetDNS(req.Service, req.Servers)

	case ipc.OpClearDNS:
		return h.runner.ClearDNS(req.Service)

	case ipc.OpFlushDNS:
		return h.runner.FlushDNS()

	case ipc.OpSaveHosts:
		if req.Path != h.allowedHostsPath {
			return fmt.Errorf("refusing to write disallowed path %q (allowed: %q)", req.Path, h.allowedHostsPath)
		}
		if err := validateHostsContent(req.Content); err != nil {
			return err
		}
		return h.runner.SaveHosts(req.Path, req.Content)

	default:
		return fmt.Errorf("unknown operation %q", req.Op)
	}
}

// parseUIDs parses a comma-separated UID list (e.g. "501,502") into a set.
// An empty string yields an empty set (only root will be authorized).
func parseUIDs(s string) (map[uint32]bool, error) {
	out := make(map[uint32]bool)
	s = strings.TrimSpace(s)
	if s == "" {
		return out, nil
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid UID %q: %w", part, err)
		}
		out[uint32(n)] = true
	}
	return out, nil
}

// validateIPs ensures every requested DNS server is a valid IP address.
func validateIPs(servers []string) error {
	for _, s := range servers {
		if net.ParseIP(s) == nil {
			return fmt.Errorf("invalid DNS server address: %q", s)
		}
	}
	return nil
}

// validateHostsContent re-parses the proposed file and validates every managed
// entry, so a compromised client cannot smuggle malformed records past root.
func validateHostsContent(content []byte) error {
	doc := hosts.Parse(content)
	for _, e := range doc.List() {
		if err := e.Validate(); err != nil {
			return fmt.Errorf("invalid managed entry: %w", err)
		}
	}
	return nil
}
