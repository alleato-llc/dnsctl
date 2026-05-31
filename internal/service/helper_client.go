package service

import "errors"

// ErrHelperUnavailable is returned by every HelperClient method until the
// privileged helper (cmd/dnsctl-helper) and its IPC are implemented.
var ErrHelperUnavailable = errors.New("privileged helper not yet implemented")

// HelperClient forwards privileged operations to the root helper daemon over
// IPC (XPC or a unix socket, TBD). It is a stub today so the GUI and a non-root
// CLI can be built against the PrivilegedRunner seam before the helper exists.
//
// When implemented, the helper must re-validate every request — it is the root
// trust boundary and must never trust its caller.
type HelperClient struct {
	// SocketPath string // future: path to the helper's listener
}

var _ PrivilegedRunner = (*HelperClient)(nil)

// NewHelperClient returns a HelperClient stub.
func NewHelperClient() *HelperClient {
	return &HelperClient{}
}

func (h *HelperClient) SetDNS(service string, servers []string) error { return ErrHelperUnavailable }
func (h *HelperClient) ClearDNS(service string) error                 { return ErrHelperUnavailable }
func (h *HelperClient) FlushDNS() error                               { return ErrHelperUnavailable }
func (h *HelperClient) SaveHosts(path string, content []byte) error   { return ErrHelperUnavailable }
