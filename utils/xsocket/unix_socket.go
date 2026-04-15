//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package xsocket

import (
	"net"
	"os"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/xerrors"
)

// GetUnixSockAddr 根据协议和原始地址，解析出结构化的 Unix Socket 地址信息。
func GetUnixSockAddr(proto, addr string) (sa unix.Sockaddr, family int, unixAddr *net.UnixAddr, err error) {
	unixAddr, err = net.ResolveUnixAddr(proto, addr)
	if err != nil {
		return
	}

	switch unixAddr.Network() {
	case "unix":
		sa, family = &unix.SockaddrUnix{Name: unixAddr.Name}, unix.AF_UNIX
	default:
		err = xerrors.ErrUnsupportedUDSProtocol
	}

	return
}

// udsSocket 创建一个用于通信的 Unix Domain Socket，并返回对应的文件描述符。
func udsSocket(proto, addr string, passive bool, sockOptInts []Option[int], sockOptStrs []Option[string]) (fd int, netAddr net.Addr, err error) {
	var (
		family int
		sa     unix.Sockaddr
	)

	if sa, family, netAddr, err = GetUnixSockAddr(proto, addr); err != nil {
		return
	}

	if fd, err = sysSocket(family, unix.SOCK_STREAM, 0); err != nil {
		err = os.NewSyscallError("socket", err)
		return
	}
	defer func() {
		if err != nil {
			// 对于非阻塞 socket 的 connect 场景，忽略 EINPROGRESS，
			// 该状态应由调用方继续处理。
			// 不过 Unix Socket 下出现 EINPROGRESS 的场景相对较少。
			if xerrors.Is(err, unix.EINPROGRESS) {
				return
			}
			_ = unix.Close(fd)
		}
	}()

	if err = execSockOpts(fd, sockOptInts); err != nil {
		return
	}
	if err = execSockOpts(fd, sockOptStrs); err != nil {
		return
	}

	if passive {
		if err = os.NewSyscallError("bind", unix.Bind(fd, sa)); err != nil {
			return
		}

		// 将 backlog 设置为系统允许的最大值。
		err = os.NewSyscallError("listen", unix.Listen(fd, listenerBacklogMaxSize))
	} else {
		err = os.NewSyscallError("connect", unix.Connect(fd, sa))
	}

	return
}
