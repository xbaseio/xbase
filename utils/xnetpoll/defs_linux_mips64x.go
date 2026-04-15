//go:build linux && poll_opt && (mips64 || mips64le)
// +build linux
// +build poll_opt
// +build mips64 mips64le

package xnetpoll

// epollevent 是针对 Linux MIPS64 架构的优化 epoll 事件结构布局。
type epollevent struct {
	events    uint32
	pad_cgo_0 [4]byte
	data      [8]byte
}
