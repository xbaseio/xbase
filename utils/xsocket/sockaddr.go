//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd
// +build darwin dragonfly freebsd linux netbsd openbsd

package xsocket

import (
	"net"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/utils/xbs"
	bsPool "github.com/xbaseio/xbase/utils/xpool/xbyteslice"
)

// NetAddrToSockaddr 将 net.Addr 转换为 Sockaddr。
// 如果输入无效或无法转换，则返回 nil。
func NetAddrToSockaddr(addr net.Addr) unix.Sockaddr {
	switch addr := addr.(type) {
	case *net.IPAddr:
		return IPAddrToSockaddr(addr)
	case *net.TCPAddr:
		return TCPAddrToSockaddr(addr)
	case *net.UDPAddr:
		return UDPAddrToSockaddr(addr)
	case *net.UnixAddr:
		sa, _ := UnixAddrToSockaddr(addr)
		return sa
	default:
		return nil
	}
}

// IPAddrToSockaddr 将 net.IPAddr 转换为 Sockaddr。
// 如果转换失败，则返回 nil。
func IPAddrToSockaddr(addr *net.IPAddr) unix.Sockaddr {
	return IPToSockaddr(addr.IP, 0, addr.Zone)
}

// TCPAddrToSockaddr 将 net.TCPAddr 转换为 Sockaddr。
// 如果转换失败，则返回 nil。
func TCPAddrToSockaddr(addr *net.TCPAddr) unix.Sockaddr {
	return IPToSockaddr(addr.IP, addr.Port, addr.Zone)
}

// UDPAddrToSockaddr 将 net.UDPAddr 转换为 Sockaddr。
// 如果转换失败，则返回 nil。
func UDPAddrToSockaddr(addr *net.UDPAddr) unix.Sockaddr {
	return IPToSockaddr(addr.IP, addr.Port, addr.Zone)
}

// IPToSockaddr 将 net.IP（可选带 IPv6 Zone）转换为 Sockaddr。
// 如果转换失败，则返回 nil。
func IPToSockaddr(ip net.IP, port int, zone string) unix.Sockaddr {
	// 未指定地址？
	if ip == nil {
		if zone != "" {
			return &unix.SockaddrInet6{Port: port, ZoneId: uint32(ip6ZoneToInt(zone))}
		}
		return &unix.SockaddrInet4{Port: port}
	}

	// 合法的 IPv4？
	if ip4 := ip.To4(); ip4 != nil && zone == "" {
		sa := unix.SockaddrInet4{Port: port}
		copy(sa.Addr[:], ip4) // 最后 4 个字节
		return &sa
	}

	// 合法的 IPv6 地址？
	if ip6 := ip.To16(); ip6 != nil {
		sa := unix.SockaddrInet6{Port: port, ZoneId: uint32(ip6ZoneToInt(zone))}
		copy(sa.Addr[:], ip6)
		return &sa
	}

	return nil
}

// UnixAddrToSockaddr 将 net.UnixAddr 转换为 Sockaddr，并返回
// 对应的类型（unix.SOCK_STREAM、unix.SOCK_DGRAM、unix.SOCK_SEQPACKET）。
// 如果转换失败，则返回 (nil, 0)。
func UnixAddrToSockaddr(addr *net.UnixAddr) (unix.Sockaddr, int) {
	t := 0
	switch addr.Net {
	case "unix":
		t = unix.SOCK_STREAM
	case "unixgram":
		t = unix.SOCK_DGRAM
	case "unixpacket":
		t = unix.SOCK_SEQPACKET
	default:
		return nil, 0
	}
	return &unix.SockaddrUnix{Name: addr.Name}, t
}

// SockaddrToTCPOrUnixAddr 将 unix.Sockaddr 转换为 net.TCPAddr 或 net.UnixAddr。
// 如果转换失败，则返回 nil。
func SockaddrToTCPOrUnixAddr(sa unix.Sockaddr) net.Addr {
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		return &net.TCPAddr{IP: sa.Addr[0:], Port: sa.Port}
	case *unix.SockaddrInet6:
		return &net.TCPAddr{IP: sa.Addr[0:], Port: sa.Port, Zone: ip6ZoneToString(sa.ZoneId)}
	case *unix.SockaddrUnix:
		return &net.UnixAddr{Name: sa.Name, Net: "unix"}
	}
	return nil
}

// SockaddrToUDPAddr 将 unix.Sockaddr 转换为 net.UDPAddr。
// 如果转换失败，则返回 nil。
func SockaddrToUDPAddr(sa unix.Sockaddr) net.Addr {
	switch sa := sa.(type) {
	case *unix.SockaddrInet4:
		return &net.UDPAddr{IP: sa.Addr[0:], Port: sa.Port}
	case *unix.SockaddrInet6:
		return &net.UDPAddr{IP: sa.Addr[0:], Port: sa.Port, Zone: ip6ZoneToString(sa.ZoneId)}
	}
	return nil
}

// ip6ZoneToInt 将 IPv6 Zone 的网络字符串转换为 unix 整数。
// 如果 zone 为空字符串，则返回 0。
func ip6ZoneToInt(zone string) int {
	if zone == "" {
		return 0
	}
	if ifi, err := net.InterfaceByName(zone); err == nil {
		return ifi.Index
	}
	n, _, _ := dtoi(zone, 0)
	return n
}

// ip6ZoneToString 将 IPv6 Zone 的 unix 整数转换为网络字符串。
// 如果 zone 为 0，则返回空字符串。
func ip6ZoneToString(zone uint32) string {
	if zone == 0 {
		return ""
	}
	if ifi, err := net.InterfaceByIndex(int(zone)); err == nil {
		return ifi.Name
	}
	return itod(uint(zone))
}

// itod 将 uint 转换为十进制字符串。
func itod(v uint) string {
	if v == 0 { // 避免字符串分配
		return "0"
	}
	// 逆序组装十进制数字。
	buf := bsPool.Get(32)
	i := len(buf) - 1
	for ; v > 0; v /= 10 {
		buf[i] = byte(v%10 + '0')
		i--
	}
	return xbs.BytesToString(buf[i:])
}

// 比我们需要的大，但又不至于大到需要担心溢出。
const big = 0xFFFFFF

// 从 s[i0] 开始将十进制字符串转换为整数。
// 返回：数字、新偏移量、是否成功。
func dtoi(s string, i0 int) (n int, i int, ok bool) {
	n = 0
	for i = i0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
		if n >= big {
			return 0, i, false
		}
	}
	if i == i0 {
		return 0, i, false
	}
	return n, i, true
}
