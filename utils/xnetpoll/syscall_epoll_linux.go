//go:build linux
// +build linux

package xnetpoll

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// epollCtl 是对 epoll_ctl 系统调用的底层封装（使用 RawSyscall6）
//
// 参数说明：
// - epfd: epoll 实例的文件描述符
// - op: 操作类型（EPOLL_CTL_ADD / EPOLL_CTL_MOD / EPOLL_CTL_DEL）
// - fd: 要操作的目标文件描述符
// - event: 事件结构体指针（epollevent）
//
// 返回：
// - 成功返回 nil
// - 失败返回 errno 对应的错误
func epollCtl(epfd int, op int, fd int, event *epollevent) error {
	// 直接调用系统调用，减少封装开销
	_, _, errno := unix.RawSyscall6(
		unix.SYS_EPOLL_CTL,
		uintptr(epfd),
		uintptr(op),
		uintptr(fd),
		uintptr(unsafe.Pointer(event)),
		0,
		0,
	)

	// errno 非 0 表示系统调用失败
	if errno != 0 {
		return errnoErr(errno)
	}

	return nil
}
