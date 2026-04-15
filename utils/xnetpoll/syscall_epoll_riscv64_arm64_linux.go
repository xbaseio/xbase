//go:build linux
// +build linux

package xnetpoll

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// epollWait 是对 epoll_wait/epoll_pwait 系统调用的底层封装
//
// 参数说明：
// - epfd: epoll 实例的文件描述符
// - events: 用于接收事件的切片
// - msec: 超时时间（毫秒）
//   - -1: 一直阻塞
//   - 0: 非阻塞
//   - >0: 最多等待指定毫秒数
//
// 返回：
// - 就绪事件数量
// - 错误信息
func epollWait(epfd int, events []epollevent, msec int) (int, error) {
	var ep unsafe.Pointer
	if len(events) > 0 {
		ep = unsafe.Pointer(&events[0])
	} else {
		ep = unsafe.Pointer(&zero)
	}

	var (
		np    uintptr
		errno unix.Errno
	)

	if msec == 0 { // 非阻塞系统调用，使用 RawSyscall6 以避免被 runtime 抢占
		np, _, errno = unix.RawSyscall6(
			unix.SYS_EPOLL_PWAIT,
			uintptr(epfd),
			uintptr(ep),
			uintptr(len(events)),
			0,
			0,
			0,
		)
	} else {
		np, _, errno = unix.Syscall6(
			unix.SYS_EPOLL_PWAIT,
			uintptr(epfd),
			uintptr(ep),
			uintptr(len(events)),
			uintptr(msec),
			0,
			0,
		)
	}

	if errno != 0 {
		return int(np), errnoErr(errno)
	}
	return int(np), nil
}
