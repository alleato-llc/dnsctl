//go:build !darwin

package main

import (
	"fmt"
	"net"
)

// peerUID is unsupported off macOS; the helper is a macOS component. This stub
// keeps `go build ./...` and CI green on other platforms.
func peerUID(conn net.Conn) (uint32, error) {
	return 0, fmt.Errorf("peer authentication is only supported on macOS")
}
