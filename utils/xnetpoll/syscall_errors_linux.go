//go:build linux
// +build linux

package xnetpoll

import "golang.org/x/sys/unix"

// 常见 Errno 错误只进行一次接口装箱，避免重复分配
var (
	errEAGAIN error = unix.EAGAIN
	errEINVAL error = unix.EINVAL
	errENOENT error = unix.ENOENT
)

// errnoErr 返回常见的 Errno 错误（已装箱），以避免运行时重复分配
func errnoErr(e unix.Errno) error {
	switch e {
	case unix.EAGAIN:
		return errEAGAIN
	case unix.EINVAL:
		return errEINVAL
	case unix.ENOENT:
		return errENOENT
	}
	return e
}

// zero 用于在某些系统调用中作为空指针占位
var zero uintptr
