//go:build darwin || netbsd || openbsd
// +build darwin netbsd openbsd

package xsocket

import (
	"syscall"

	"golang.org/x/sys/unix"
)

func sysSocket(family, sotype, proto int) (fd int, err error) {
	syscall.ForkLock.RLock()
	if fd, err = unix.Socket(family, sotype, proto); err == nil {
		unix.CloseOnExec(fd)
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		return
	}
	if err = unix.SetNonblock(fd, true); err != nil {
		_ = unix.Close(fd)
	}
	return
}

func sysAccept(fd int) (nfd int, sa unix.Sockaddr, err error) {
	syscall.ForkLock.RLock()
	if nfd, sa, err = unix.Accept(fd); err == nil {
		unix.CloseOnExec(nfd)
	}
	syscall.ForkLock.RUnlock()
	if err != nil {
		return
	}
	if err = unix.SetNonblock(nfd, true); err != nil {
		_ = unix.Close(nfd)
	}
	return
}
