package xio

import "golang.org/x/sys/unix"

// Writev 在 Linux 上调用 writev() 系统调用。
func Writev(fd int, iov [][]byte) (int, error) {
	if len(iov) == 0 {
		return 0, nil
	}
	return unix.Writev(fd, iov)
}

// Readv 在 Linux 上调用 readv() 系统调用。
func Readv(fd int, iov [][]byte) (int, error) {
	if len(iov) == 0 {
		return 0, nil
	}
	return unix.Readv(fd, iov)
}
