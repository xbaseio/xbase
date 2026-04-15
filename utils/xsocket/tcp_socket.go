//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd
// +build darwin dragonfly freebsd linux netbsd openbsd

package xsocket

import (
	"errors"
	"net"
	"os"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/xerrors"
)

var listenerBacklogMaxSize = maxListenerBacklog()

// GetTCPSockAddr 根据协议和原始地址解析出结构化地址信息。
func GetTCPSockAddr(proto, addr string) (sa unix.Sockaddr, family int, tcpAddr *net.TCPAddr, ipv6only bool, err error) {
	var tcpVersion string

	tcpAddr, err = net.ResolveTCPAddr(proto, addr)
	if err != nil {
		return
	}

	tcpVersion, err = determineTCPProto(proto, tcpAddr)
	if err != nil {
		return
	}

	switch tcpVersion {
	case "tcp4":
		family = unix.AF_INET
		sa, err = ipToSockaddr(family, tcpAddr.IP, tcpAddr.Port, "")

	case "tcp6":
		ipv6only = true
		fallthrough

	case "tcp":
		family = unix.AF_INET6
		sa, err = ipToSockaddr(family, tcpAddr.IP, tcpAddr.Port, tcpAddr.Zone)

	default:
		err = xerrors.ErrUnsupportedProtocol
	}

	return
}

func determineTCPProto(proto string, addr *net.TCPAddr) (string, error) {
	// 如果协议为 "tcp"，则根据解析出的 IP 地址类型自动判断版本。
	// 否则直接使用调用方传入的协议。

	if addr.IP.To4() != nil {
		return "tcp4", nil
	}

	if addr.IP.To16() != nil {
		return "tcp6", nil
	}

	switch proto {
	case "tcp", "tcp4", "tcp6":
		return proto, nil
	}

	return "", xerrors.ErrUnsupportedTCPProtocol
}

// tcpSocket 创建一个通信端点，并返回对应的文件描述符。
func tcpSocket(proto, addr string, passive bool, sockOptInts []Option[int], sockOptStrs []Option[string]) (fd int, netAddr net.Addr, err error) {
	var (
		family   int
		ipv6only bool
		sa       unix.Sockaddr
	)

	if sa, family, netAddr, ipv6only, err = GetTCPSockAddr(proto, addr); err != nil {
		return
	}

	if fd, err = sysSocket(family, unix.SOCK_STREAM, unix.IPPROTO_TCP); err != nil {
		err = os.NewSyscallError("socket", err)
		return
	}

	defer func() {
		if err != nil {
			// 非阻塞 connect 时返回 EINPROGRESS，这是正常情况，由调用方处理
			if errors.Is(err, unix.EINPROGRESS) {
				return
			}
			_ = unix.Close(fd)
		}
	}()

	if family == unix.AF_INET6 && ipv6only {
		if err = SetIPv6Only(fd, 1); err != nil {
			return
		}
	}

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
		// backlog 设置为系统允许的最大值
		err = os.NewSyscallError("listen", unix.Listen(fd, listenerBacklogMaxSize))
	} else {
		err = os.NewSyscallError("connect", unix.Connect(fd, sa))
	}

	return
}
