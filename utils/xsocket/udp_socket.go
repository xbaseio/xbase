//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package xsocket

import (
	"net"
	"os"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/xerrors"
)

// GetUDPSockAddr 根据协议和原始地址，解析出结构化的 UDP 地址信息。
func GetUDPSockAddr(proto, addr string) (sa unix.Sockaddr, family int, udpAddr *net.UDPAddr, ipv6only bool, err error) {
	var udpVersion string

	udpAddr, err = net.ResolveUDPAddr(proto, addr)
	if err != nil {
		return
	}

	udpVersion, err = determineUDPProto(proto, udpAddr)
	if err != nil {
		return
	}

	switch udpVersion {
	case "udp4":
		family = unix.AF_INET
		sa, err = ipToSockaddr(family, udpAddr.IP, udpAddr.Port, "")
	case "udp6":
		ipv6only = true
		fallthrough
	case "udp":
		family = unix.AF_INET6
		sa, err = ipToSockaddr(family, udpAddr.IP, udpAddr.Port, udpAddr.Zone)
	default:
		err = xerrors.ErrUnsupportedProtocol
	}

	return
}

func determineUDPProto(proto string, addr *net.UDPAddr) (string, error) {
	// 如果协议传入的是 "udp"，则尝试根据解析后的 IP 地址长度
	// 自动判断实际使用的是 udp4 还是 udp6。
	// 否则，直接使用调用方传入的协议类型。

	if addr.IP.To4() != nil {
		return "udp4", nil
	}

	if addr.IP.To16() != nil {
		return "udp6", nil
	}

	switch proto {
	case "udp", "udp4", "udp6":
		return proto, nil
	}

	return "", xerrors.ErrUnsupportedUDPProtocol
}

// udpSocket 创建一个用于通信的 UDP 套接字，并返回对应的文件描述符。
func udpSocket(proto, addr string, connect bool, sockOptInts []Option[int], sockOptStrs []Option[string]) (fd int, netAddr net.Addr, err error) {
	var (
		family   int
		ipv6only bool
		sa       unix.Sockaddr
	)

	if sa, family, netAddr, ipv6only, err = GetUDPSockAddr(proto, addr); err != nil {
		return
	}

	if fd, err = sysSocket(family, unix.SOCK_DGRAM, unix.IPPROTO_UDP); err != nil {
		err = os.NewSyscallError("socket", err)
		return
	}
	defer func() {
		if err != nil {
			// 对于非阻塞 socket 的 connect 场景，忽略 EINPROGRESS，
			// 该状态应由调用方继续处理。
			if xerrors.Is(err, unix.EINPROGRESS) {
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

	// 允许广播。
	if err = os.NewSyscallError("setsockopt", unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_BROADCAST, 1)); err != nil {
		return
	}

	if err = execSockOpts(fd, sockOptInts); err != nil {
		return
	}
	if err = execSockOpts(fd, sockOptStrs); err != nil {
		return
	}

	if connect {
		err = os.NewSyscallError("connect", unix.Connect(fd, sa))
	} else {
		err = os.NewSyscallError("bind", unix.Bind(fd, sa))
	}

	return
}
