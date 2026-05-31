package main

import (
	"os"

	"github.com/nycjv321/dnsctl/internal/service"
)

// chooseRunner selects how privileged operations are performed:
//   - running as root (e.g. `sudo dnsctl`): do the work in-process;
//   - otherwise: forward to the dnsctl-helper daemon over IPC.
//
// Reads never need privileges, so non-root `dnsctl hosts list` and the TUI
// status view work regardless of whether the helper is installed. If a
// privileged write is attempted without root and without the helper, the
// HelperClient surfaces a clear connection error.
func chooseRunner() service.PrivilegedRunner {
	if os.Geteuid() == 0 {
		return service.NewDirectRunner()
	}
	client := service.NewHelperClient()
	// Allow overriding the helper socket (GUI sandboxes, tests, dev setups).
	if sock := os.Getenv("DNSCTL_HELPER_SOCKET"); sock != "" {
		client.SocketPath = sock
	}
	return client
}
