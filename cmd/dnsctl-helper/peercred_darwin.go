//go:build darwin

package main

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

// peerUID returns the effective UID of the process on the other end of a unix
// socket connection, via the macOS LOCAL_PEERCRED socket option. This is the
// basis for the helper's authorization decision.
func peerUID(conn net.Conn) (uint32, error) {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return 0, fmt.Errorf("connection is not a unix socket")
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf("syscall conn: %w", err)
	}

	var cred *unix.Xucred
	var sockErr error
	if err := raw.Control(func(fd uintptr) {
		cred, sockErr = unix.GetsockoptXucred(int(fd), unix.SOL_LOCAL, unix.LOCAL_PEERCRED)
	}); err != nil {
		return 0, fmt.Errorf("control fd: %w", err)
	}
	if sockErr != nil {
		return 0, fmt.Errorf("getsockopt LOCAL_PEERCRED: %w", sockErr)
	}
	return cred.Uid, nil
}
