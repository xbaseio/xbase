//go:build linux && poll_opt && (mips || mipsle)
// +build linux
// +build poll_opt
// +build mips mipsle

package xnetpoll

// epollevent 是针对 Linux MIPS 32 位架构的优化 epoll 事件结构布局。
type epollevent struct {
	events    uint32
	pad_cgo_0 [4]byte
	data      uint64
}
