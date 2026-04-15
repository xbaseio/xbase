//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd
// +build darwin dragonfly freebsd linux netbsd openbsd

// Package xsocket 提供一些实用的 socket 相关函数。
package xsocket

import (
	"net"

	"golang.org/x/sys/unix"
)

// Option 用于设置 socket 选项。
type Option[T int | string] struct {
	SetSockOpt func(int, T) error
	Opt        T
}

func execSockOpts[T int | string](fd int, opts []Option[T]) error {
	for _, opt := range opts {
		if err := opt.SetSockOpt(fd, opt.Opt); err != nil {
			return err
		}
	}
	return nil
}

// TCPSocket 创建一个 TCP socket，并返回对应的文件描述符。
// 传入的 socket 选项会设置到返回的文件描述符上。
func TCPSocket(proto, addr string, passive bool, sockOptInts []Option[int], sockOptStrs []Option[string]) (int, net.Addr, error) {
	return tcpSocket(proto, addr, passive, sockOptInts, sockOptStrs)
}

// UDPSocket 创建一个 UDP socket，并返回对应的文件描述符。
// 传入的 socket 选项会设置到返回的文件描述符上。
func UDPSocket(proto, addr string, connect bool, sockOptInts []Option[int], sockOptStrs []Option[string]) (int, net.Addr, error) {
	return udpSocket(proto, addr, connect, sockOptInts, sockOptStrs)
}

// UnixSocket 创建一个 Unix socket，并返回对应的文件描述符。
// 传入的 socket 选项会设置到返回的文件描述符上。
func UnixSocket(proto, addr string, passive bool, sockOptInts []Option[int], sockOptStrs []Option[string]) (int, net.Addr, error) {
	return udsSocket(proto, addr, passive, sockOptInts, sockOptStrs)
}

// Accept 接受下一个传入的 socket，
// 并同时为其设置 O_NONBLOCK 和 O_CLOEXEC 标志位。
func Accept(fd int) (int, unix.Sockaddr, error) {
	return sysAccept(fd)
}
