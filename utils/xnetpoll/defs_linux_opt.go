//go:build linux && poll_opt && !(mips || mipsle || mips64 || mips64le)
// +build linux,poll_opt,!mips,!mipsle,!mips64,!mips64le

package xnetpoll

// epollevent 是针对大多数 Linux 架构的优化 epoll 事件结构布局。
type epollevent struct {
	events uint32
	_pad   uint32
	data   [8]byte
}
