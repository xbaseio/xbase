//go:build dragonfly || freebsd || linux || netbsd
// +build dragonfly freebsd linux netbsd

package xsocket

import (
	"os"

	"github.com/xbaseio/xbase/xerrors"
	"golang.org/x/sys/unix"
)

// SetKeepAlivePeriod 启用 SO_KEEPALIVE 选项，并设置：
// TCP_KEEPIDLE/TCP_KEEPALIVE 为指定秒数，
// TCP_KEEPCNT 为 5，
// TCP_KEEPINTVL 为 secs/5。
func SetKeepAlivePeriod(fd, secs int) error {
	if secs <= 0 {
		return xerrors.New("invalid time duration")
	}

	interval := secs / 5
	if interval == 0 {
		interval = 1
	}

	return SetKeepAlive(fd, true, secs, interval, 5)
}

// SetKeepAlive 启用/禁用 socket 的 TCP keepalive 功能。
func SetKeepAlive(fd int, enabled bool, idle, intvl, cnt int) error {
	if enabled && (idle <= 0 || intvl <= 0 || cnt <= 0) {
		return xerrors.New("invalid time duration")
	}

	var on int
	if enabled {
		on = 1
	}

	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_KEEPALIVE, on); err != nil {
		return os.NewSyscallError("setsockopt", err)
	}

	if !enabled {
		// 如果关闭 keepalive，则忽略 TCP_KEEP* 相关选项。
		return nil
	}

	if err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPIDLE, idle); err != nil {
		return os.NewSyscallError("setsockopt", err)
	}

	if err := unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPINTVL, intvl); err != nil {
		return os.NewSyscallError("setsockopt", err)
	}

	return os.NewSyscallError("setsockopt",
		unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_KEEPCNT, cnt))
}
