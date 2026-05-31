// Command dnsctl-helper is the privileged root daemon that performs dnsctl's
// root-only operations on behalf of unprivileged clients (the GUI, a non-root
// CLI). Clients talk to it over a unix socket using the internal/ipc protocol.
//
// SECURITY: this process runs as root and is the trust boundary. It must
// re-validate and authorize every request and never trust its caller:
//   - hosts writes are restricted to an allow-listed path and the content is
//     re-parsed and validated;
//   - DNS server values are validated as IPs.
//
// TODO(security): authenticate the connecting peer's UID before acting
// (LOCAL_PEERCRED on darwin) so only permitted users can drive the helper.
// Until that lands, rely on the 0600 socket permissions.
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/nycjv321/dnsctl/internal/hosts"
	"github.com/nycjv321/dnsctl/internal/ipc"
	"github.com/nycjv321/dnsctl/internal/service"
)

func main() {
	socket := flag.String("socket", ipc.DefaultSocketPath, "unix socket path to listen on")
	hostsPath := flag.String("hosts-file", hosts.DefaultPath, "the only path hosts writes are permitted to target")
	flag.Parse()

	// Remove any stale socket from a previous run before binding.
	if err := os.Remove(*socket); err != nil && !os.IsNotExist(err) {
		log.Fatalf("remove stale socket %s: %v", *socket, err)
	}
	ln, err := net.Listen("unix", *socket)
	if err != nil {
		log.Fatalf("listen on %s: %v", *socket, err)
	}
	defer ln.Close()
	if err := os.Chmod(*socket, 0600); err != nil {
		log.Fatalf("chmod socket: %v", err)
	}

	h := &handler{runner: service.NewDirectRunner(), allowedHostsPath: *hostsPath}
	log.Printf("dnsctl-helper listening on %s (hosts writes limited to %s)", *socket, *hostsPath)
	if err := serve(ln, h); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// handler dispatches validated requests to the privileged runner.
type handler struct {
	runner           service.PrivilegedRunner
	allowedHostsPath string
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

// handle reads one request, dispatches it, and writes one response.
func (h *handler) handle(conn net.Conn) {
	defer conn.Close()

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

// dispatch validates and executes a single request. This is the enforcement
// point for the trust boundary.
func (h *handler) dispatch(req ipc.Request) error {
	switch req.Op {
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
