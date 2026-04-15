//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package xio

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// Writev 直接调用 writev 系统调用。
//
// 注意：Darwin 上的 SYS_WRITEV 未来可能会被弃用，
// Go 官方更建议使用 libSystem 封装而不是直接发起系统调用，
// 因此，这种 writev 实现方式将来可能不再向后兼容。
func Writev(fd int, bs [][]byte) (int, error) {
	if len(bs) == 0 {
		return 0, nil
	}

	iov := bytes2iovec(bs)
	n, _, err := unix.RawSyscall(
		unix.SYS_WRITEV,
		uintptr(fd),
		uintptr(unsafe.Pointer(&iov[0])),
		uintptr(len(iov)),
	) //nolint:staticcheck

	if err != 0 {
		return int(n), err
	}
	return int(n), nil
}

// Readv 直接调用 readv 系统调用。
//
// 注意：Darwin 上的 SYS_READV 未来可能会被弃用，
// Go 官方更建议使用 libSystem 封装而不是直接发起系统调用，
// 因此，这种 readv 实现方式将来可能不再向后兼容。
func Readv(fd int, bs [][]byte) (int, error) {
	if len(bs) == 0 {
		return 0, nil
	}

	iov := bytes2iovec(bs)

	// 系统调用
	n, _, err := unix.RawSyscall(
		unix.SYS_READV,
		uintptr(fd),
		uintptr(unsafe.Pointer(&iov[0])),
		uintptr(len(iov)),
	) //nolint:staticcheck

	if err != 0 {
		return int(n), err
	}
	return int(n), nil
}

var _zero uintptr

// bytes2iovec 将 [][]byte 转换为 []unix.Iovec。
func bytes2iovec(bs [][]byte) []unix.Iovec {
	iovecs := make([]unix.Iovec, len(bs))
	for i, b := range bs {
		iovecs[i].SetLen(len(b))
		if len(b) > 0 {
			iovecs[i].Base = &b[0]
		} else {
			iovecs[i].Base = (*byte)(unsafe.Pointer(&_zero))
		}
	}
	return iovecs
}
