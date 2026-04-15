//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd
// +build darwin dragonfly freebsd linux netbsd openbsd

package xsocket

import (
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

func ipToSockaddrInet4(ip net.IP, port int) (unix.SockaddrInet4, error) {
	if len(ip) == 0 {
		ip = net.IPv4zero
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return unix.SockaddrInet4{}, &net.AddrError{Err: "non-IPv4 address", Addr: ip.String()}
	}
	sa := unix.SockaddrInet4{Port: port}
	copy(sa.Addr[:], ip4)
	return sa, nil
}

func ipToSockaddrInet6(ip net.IP, port int, zone string) (unix.SockaddrInet6, error) {
	// 一般来说，IP 通配地址，也就是 "0.0.0.0" 或 "::"，表示整个 IP 地址空间。
	// 由于一些历史原因，在某些 IP 节点操作中，它也被用来表示“任意可用地址”。
	//
	// 当 IP 节点支持 IPv4-mapped IPv6 地址时，
	// 我们允许监听器通过指定 IPv6 通配地址，
	// 同时监听 IPv4 和 IPv6 两个地址空间的通配地址。
	if len(ip) == 0 || ip.Equal(net.IPv4zero) {
		ip = net.IPv6zero
	}
	// 我们接受任意 IPv6 地址，包括 IPv4-mapped IPv6 地址。
	ip6 := ip.To16()
	if ip6 == nil {
		return unix.SockaddrInet6{}, &net.AddrError{Err: "non-IPv6 address", Addr: ip.String()}
	}

	sa := unix.SockaddrInet6{Port: port}
	copy(sa.Addr[:], ip6)

	iface, err := net.InterfaceByName(zone)
	if err != nil {
		return sa, nil
	}
	sa.ZoneId = uint32(iface.Index)

	return sa, nil
}

func ipToSockaddr(family int, ip net.IP, port int, zone string) (unix.Sockaddr, error) {
	switch family {
	case syscall.AF_INET:
		sa, err := ipToSockaddrInet4(ip, port)
		if err != nil {
			return nil, err
		}
		return &sa, nil

	case syscall.AF_INET6:
		sa, err := ipToSockaddrInet6(ip, port, zone)
		if err != nil {
			return nil, err
		}
		return &sa, nil
	}

	return nil, &net.AddrError{Err: "invalid address family", Addr: ip.String()}
}
