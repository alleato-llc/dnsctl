package service

import (
	"errors"
	"fmt"
	"net"

	"github.com/nycjv321/dnsctl/internal/ipc"
)

// HelperClient forwards privileged operations to the root helper daemon
// (cmd/dnsctl-helper) over a unix socket. It lets the unprivileged GUI and a
// non-root CLI perform privileged work without being root themselves.
//
// The helper is the root trust boundary: it re-validates and authorizes every
// request, so it must never assume the client is well-behaved.
type HelperClient struct {
	// SocketPath is the helper's unix socket. Empty means ipc.DefaultSocketPath.
	SocketPath string
}

var _ PrivilegedRunner = (*HelperClient)(nil)

// NewHelperClient returns a HelperClient using the default socket path.
func NewHelperClient() *HelperClient {
	return &HelperClient{SocketPath: ipc.DefaultSocketPath}
}

func (h *HelperClient) socketPath() string {
	if h.SocketPath == "" {
		return ipc.DefaultSocketPath
	}
	return h.SocketPath
}

// call sends one request and returns the helper's reported error, if any.
func (h *HelperClient) call(req ipc.Request) error {
	conn, err := net.Dial("unix", h.socketPath())
	if err != nil {
		return fmt.Errorf("connect to dnsctl-helper at %s: %w", h.socketPath(), err)
	}
	defer conn.Close()

	if err := ipc.WriteRequest(conn, req); err != nil {
		return fmt.Errorf("send request to helper: %w", err)
	}
	var resp ipc.Response
	if err := ipc.ReadResponse(conn, &resp); err != nil {
		return fmt.Errorf("read response from helper: %w", err)
	}
	if resp.Error != "" {
		return errors.New(resp.Error)
	}
	return nil
}

// Ping checks that the helper is reachable and authorizes this client.
func (h *HelperClient) Ping() error {
	return h.call(ipc.Request{Op: ipc.OpPing})
}

func (h *HelperClient) SetDNS(service string, servers []string) error {
	return h.call(ipc.Request{Op: ipc.OpSetDNS, Service: service, Servers: servers})
}

func (h *HelperClient) ClearDNS(service string) error {
	return h.call(ipc.Request{Op: ipc.OpClearDNS, Service: service})
}

func (h *HelperClient) FlushDNS() error {
	return h.call(ipc.Request{Op: ipc.OpFlushDNS})
}

func (h *HelperClient) SaveHosts(path string, content []byte) error {
	return h.call(ipc.Request{Op: ipc.OpSaveHosts, Path: path, Content: content})
}
