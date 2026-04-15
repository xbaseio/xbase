//go:build !poll_opt

package xnetpoll

import "golang.org/x/sys/unix"

// epollevent 表示 epoll 事件结构（未启用优化版本，直接使用系统定义）。
type epollevent = unix.EpollEvent
