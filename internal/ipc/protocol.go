// Package ipc defines the request/response protocol shared between the
// privileged helper daemon (cmd/dnsctl-helper) and the service.HelperClient
// that talks to it. Messages are newline-delimited JSON over a unix socket:
// one request and one response per connection.
package ipc

import (
	"encoding/json"
	"io"
)

// DefaultSocketPath is where the helper listens by default. It lives under a
// root-writable directory; the socket is created with 0600 so only root (the
// helper) and processes it explicitly authorizes can use it.
const DefaultSocketPath = "/var/run/dnsctl-helper.sock"

// Op identifies a privileged operation.
type Op string

const (
	OpSetDNS    Op = "set_dns"
	OpClearDNS  Op = "clear_dns"
	OpFlushDNS  Op = "flush_dns"
	OpSaveHosts Op = "save_hosts"
)

// Request is a single privileged operation. Only the fields relevant to Op are
// populated.
type Request struct {
	Op      Op       `json:"op"`
	Service string   `json:"service,omitempty"`
	Servers []string `json:"servers,omitempty"`
	Path    string   `json:"path,omitempty"`
	Content []byte   `json:"content,omitempty"`
}

// Response carries the outcome. Error is empty on success.
type Response struct {
	Error string `json:"error,omitempty"`
}

// WriteRequest encodes a request to w.
func WriteRequest(w io.Writer, req Request) error {
	return json.NewEncoder(w).Encode(req)
}

// ReadRequest decodes a single request from r.
func ReadRequest(r io.Reader, req *Request) error {
	return json.NewDecoder(r).Decode(req)
}

// WriteResponse encodes a response to w.
func WriteResponse(w io.Writer, resp Response) error {
	return json.NewEncoder(w).Encode(resp)
}

// ReadResponse decodes a single response from r.
func ReadResponse(r io.Reader, resp *Response) error {
	return json.NewDecoder(r).Decode(resp)
}
