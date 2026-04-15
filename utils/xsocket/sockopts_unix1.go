//go:build darwin || dragonfly || linux || netbsd || openbsd
// +build darwin dragonfly linux netbsd openbsd

package xsocket

import (
	"os"

	"golang.org/x/sys/unix"
)

// SetReuseport 在 socket 上启用 SO_REUSEPORT 选项。
func SetReuseport(fd, reusePort int) error {
	return os.NewSyscallError("setsockopt",
		unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, reusePort))
}
