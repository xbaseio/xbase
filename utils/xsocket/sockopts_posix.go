//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd
// +build darwin dragonfly freebsd linux netbsd openbsd

package xsocket

import (
	"net"
	"os"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/xerrors"
)

// SetNoDelay 控制操作系统是否延迟发送数据包（Nagle 算法）。
//
// 默认值为 true（不延迟），表示在调用 Write 后尽快发送数据。
func SetNoDelay(fd, noDelay int) error {
	return os.NewSyscallError("setsockopt",
		unix.SetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_NODELAY, noDelay))
}

// SetRecvBuffer 设置 socket 的接收缓冲区大小。
func SetRecvBuffer(fd, size int) error {
	return os.NewSyscallError("setsockopt",
		unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, size))
}

// SetSendBuffer 设置 socket 的发送缓冲区大小。
func SetSendBuffer(fd, size int) error {
	return os.NewSyscallError("setsockopt",
		unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_SNDBUF, size))
}

// SetReuseAddr 在 socket 上启用 SO_REUSEADDR 选项。
func SetReuseAddr(fd, reuseAddr int) error {
	return os.NewSyscallError("setsockopt",
		unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, reuseAddr))
}

// SetIPv6Only 控制 IPv6 socket 只处理 IPv6 请求，或同时处理 IPv4 和 IPv6 请求。
func SetIPv6Only(fd, ipv6only int) error {
	return os.NewSyscallError("setsockopt",
		unix.SetsockoptInt(fd, unix.IPPROTO_IPV6, unix.IPV6_V6ONLY, ipv6only))
}

// SetLinger 设置在连接仍有数据待发送或待确认时，Close 的行为。
//
// 如果 sec < 0（默认）：操作系统在后台完成数据发送。
// 如果 sec == 0：操作系统丢弃未发送或未确认的数据。
// 如果 sec > 0：行为类似 sec < 0，但在部分系统中超过 sec 秒后，
//
//	仍未发送的数据可能被丢弃。
func SetLinger(fd, sec int) error {
	var l unix.Linger
	if sec >= 0 {
		l.Onoff = 1
		l.Linger = int32(sec)
	} else {
		l.Onoff = 0
		l.Linger = 0
	}
	return os.NewSyscallError("setsockopt",
		unix.SetsockoptLinger(fd, syscall.SOL_SOCKET, syscall.SO_LINGER, &l))
}

// SetMulticastMembership 根据 IP 版本返回一个用于设置组播成员关系的函数。
// 如果无法应用组播，则返回 nil。
func SetMulticastMembership(proto string, udpAddr *net.UDPAddr) func(int, int) error {
	udpVersion, err := determineUDPProto(proto, udpAddr)
	if err != nil {
		return nil
	}

	switch udpVersion {
	case "udp4":
		return func(fd int, ifIndex int) error {
			return SetIPv4MulticastMembership(fd, udpAddr.IP, ifIndex)
		}
	case "udp6":
		return func(fd int, ifIndex int) error {
			return SetIPv6MulticastMembership(fd, udpAddr.IP, ifIndex)
		}
	default:
		return nil
	}
}

// SetIPv4MulticastMembership 将 fd 加入指定的 IPv4 组播地址。
// ifIndex 是接收组播数据的网卡索引。
// 如果 ifIndex 为 0，则由操作系统选择默认网卡（多网卡环境常用）。
func SetIPv4MulticastMembership(fd int, mcast net.IP, ifIndex int) error {
	// IPv4 下通过 IP 地址选择组播接口（IPv6 使用索引）
	ip, err := interfaceFirstIPv4Addr(ifIndex)
	if err != nil {
		return err
	}

	mreq := &unix.IPMreq{}
	copy(mreq.Multiaddr[:], mcast.To4())
	copy(mreq.Interface[:], ip.To4())

	if ifIndex > 0 {
		if err := os.NewSyscallError("setsockopt",
			unix.SetsockoptInet4Addr(fd, syscall.IPPROTO_IP, syscall.IP_MULTICAST_IF, mreq.Interface)); err != nil {
			return err
		}
	}

	if err := os.NewSyscallError("setsockopt",
		unix.SetsockoptByte(fd, syscall.IPPROTO_IP, syscall.IP_MULTICAST_LOOP, 0)); err != nil {
		return err
	}

	return os.NewSyscallError("setsockopt",
		unix.SetsockoptIPMreq(fd, syscall.IPPROTO_IP, syscall.IP_ADD_MEMBERSHIP, mreq))
}

// SetIPv6MulticastMembership 将 fd 加入指定的 IPv6 组播地址。
// ifIndex 是接收组播数据的网卡索引。
// 如果 ifIndex 为 0，则由操作系统选择默认网卡。
func SetIPv6MulticastMembership(fd int, mcast net.IP, ifIndex int) error {
	mreq := &unix.IPv6Mreq{}
	mreq.Interface = uint32(ifIndex)
	copy(mreq.Multiaddr[:], mcast.To16())

	if ifIndex > 0 {
		if err := os.NewSyscallError("setsockopt",
			unix.SetsockoptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_IF, ifIndex)); err != nil {
			return err
		}
	}

	if err := os.NewSyscallError("setsockopt",
		unix.SetsockoptInt(fd, syscall.IPPROTO_IPV6, syscall.IPV6_MULTICAST_LOOP, 0)); err != nil {
		return err
	}

	return os.NewSyscallError("setsockopt",
		unix.SetsockoptIPv6Mreq(fd, syscall.IPPROTO_IPV6, syscall.IPV6_JOIN_GROUP, mreq))
}

// interfaceFirstIPv4Addr 返回指定网卡的第一个 IPv4 地址。
func interfaceFirstIPv4Addr(ifIndex int) (net.IP, error) {
	if ifIndex == 0 {
		return net.IP([]byte{0, 0, 0, 0}), nil
	}

	iface, err := net.InterfaceByIndex(ifIndex)
	if err != nil {
		return nil, err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	for _, addr := range addrs {
		ip, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			return nil, err
		}
		if ip.To4() != nil {
			return ip, nil
		}
	}

	return nil, xerrors.ErrNoIPv4AddressOnInterface
}
