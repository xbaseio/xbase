//go:build darwin || dragonfly || freebsd || netbsd || openbsd
// +build darwin dragonfly freebsd netbsd openbsd

package xsocket

import (
	"runtime"

	"golang.org/x/sys/unix"
)

func maxListenerBacklog() int {
	var (
		n   uint32
		err error
	)

	switch runtime.GOOS {
	case "darwin":
		n, err = unix.SysctlUint32("kern.ipc.somaxconn")
	case "freebsd":
		n, err = unix.SysctlUint32("kern.ipc.soacceptqueue")
	}

	if n == 0 || err != nil {
		return unix.SOMAXCONN
	}

	// FreeBSD 使用 uint16 存储 backlog，Linux 也是如此。
	// 假设其他 BSD 系统同样如此。这里对值进行截断以避免溢出回绕。
	// 参考 issue 5030。
	if n > 1<<16-1 {
		n = 1<<16 - 1
	}

	return int(n)
}
