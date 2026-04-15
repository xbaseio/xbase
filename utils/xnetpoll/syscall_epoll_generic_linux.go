//go:build !arm64 && !riscv64 && poll_opt
// +build !arm64,!riscv64,poll_opt

package xnetpoll

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

// epollWait 等待 epoll 事件。
//
// 参数说明：
//   - epfd: epoll 文件描述符
//   - events: 用于接收事件的缓冲区
//   - msec: 等待超时时间（毫秒）
//   - msec == 0: 非阻塞调用
//   - msec  > 0: 最多阻塞 msec 毫秒
//   - msec  < 0: 一直阻塞直到有事件
//
// 返回值：
//   - n: 实际返回的事件数量
//   - err: 系统调用错误
func epollWait(epfd int, events []epollevent, msec int) (n int, err error) {
	var eventPtr unsafe.Pointer
	if len(events) > 0 {
		eventPtr = unsafe.Pointer(&events[0])
	} else {
		eventPtr = unsafe.Pointer(&zero)
	}

	var (
		r1    uintptr
		errno unix.Errno
	)

	// 非阻塞调用时使用 RawSyscall6，尽量避免 runtime 抢占带来的额外开销。
	if msec == 0 {
		r1, _, errno = unix.RawSyscall6(
			unix.SYS_EPOLL_WAIT,
			uintptr(epfd),
			uintptr(eventPtr),
			uintptr(len(events)),
			0,
			0,
			0,
		)
	} else {
		r1, _, errno = unix.Syscall6(
			unix.SYS_EPOLL_WAIT,
			uintptr(epfd),
			uintptr(eventPtr),
			uintptr(len(events)),
			uintptr(msec),
			0,
			0,
		)
	}

	if errno != 0 {
		return int(r1), errnoErr(errno)
	}
	return int(r1), nil
}
