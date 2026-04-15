//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd
// +build darwin dragonfly freebsd linux netbsd openbsd

package xsocket

import (
	"sync/atomic"
	"syscall"

	"golang.org/x/sys/unix"
)

// Dup 复制给定的 fd，并将其标记为 close-on-exec。
func Dup(fd int) (int, error) {
	return dupCloseOnExec(fd)
}

// tryDupCloexec 表示是否尝试使用 F_DUPFD_CLOEXEC。
// 如果内核不支持，会将其设置为 false。
var tryDupCloexec atomic.Bool

func init() {
	tryDupCloexec.Store(true)
}

// dupCloseOnExec 复制给定的 fd，并将其标记为 close-on-exec。
func dupCloseOnExec(fd int) (int, error) {
	if tryDupCloexec.Load() {
		r, err := unix.FcntlInt(uintptr(fd), unix.F_DUPFD_CLOEXEC, 0)
		if err == nil {
			return r, nil
		}
		switch err.(syscall.Errno) {
		case unix.EINVAL, unix.ENOSYS:
			// 旧内核，或者 js/wasm（会返回 ENOSYS）。
			// 从现在开始退回到可移植的老方式。
			tryDupCloexec.Store(false)
		default:
			return -1, err
		}
	}
	return dupCloseOnExecOld(fd)
}

// dupCloseOnExecOld 是传统方式：先 dup 一个 fd，
// 再通过两次系统调用设置它的 O_CLOEXEC 标志位。
func dupCloseOnExecOld(fd int) (int, error) {
	syscall.ForkLock.RLock()
	defer syscall.ForkLock.RUnlock()

	newFD, err := syscall.Dup(fd)
	if err != nil {
		return -1, err
	}

	syscall.CloseOnExec(newFD)
	return newFD, nil
}
