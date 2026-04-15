//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd
// +build darwin dragonfly freebsd linux netbsd openbsd

package xnetpoll

// PollEventHandler 表示 poller 触发 I/O 事件时的回调函数。
// 参数说明：
//
//	fd: 文件描述符
//	event: 事件类型（读/写等）
//	flags: 附加标志（错误等）
type PollEventHandler func(fd int, event IOEvent, flags IOFlags) error

// PollAttachment 表示绑定到 poller 的用户数据。
// 在 epoll 中对应 epoll_data，在 kqueue 中对应 udata。
type PollAttachment struct {
	FD       int
	Callback PollEventHandler
}
