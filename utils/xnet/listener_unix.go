//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package xnet

import (
	"net"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xnetpoll"
	"github.com/xbaseio/xbase/utils/xsocket"
	"github.com/xbaseio/xbase/xerrors"
	"golang.org/x/sys/unix"
)

// listener 表示一个监听器。
type listener struct {
	openOnce, closeOnce sync.Once

	fd               int
	addr             net.Addr
	address, network string

	sockOptInts []xsocket.Option[int]
	sockOptStrs []xsocket.Option[string]

	pollAttachment *xnetpoll.PollAttachment // 监听器绑定到 poller 的附件
}

// packPollAttachment 打包 poller 附件。
func (ln *listener) packPollAttachment(handler xnetpoll.PollEventHandler) *xnetpoll.PollAttachment {
	ln.pollAttachment = &xnetpoll.PollAttachment{
		FD:       ln.fd,
		Callback: handler,
	}
	return ln.pollAttachment
}

// dup 复制监听 fd。
func (ln *listener) dup() (int, error) {
	return xsocket.Dup(ln.fd)
}

// open 打开监听器。
func (ln *listener) open() (err error) {
	ln.openOnce.Do(func() {
		switch ln.network {
		case "tcp", "tcp4", "tcp6":
			ln.fd, ln.addr, err = xsocket.TCPSocket(
				ln.network,
				ln.address,
				true,
				ln.sockOptInts,
				ln.sockOptStrs,
			)
			ln.network = "tcp"

		case "udp", "udp4", "udp6":
			ln.fd, ln.addr, err = xsocket.UDPSocket(
				ln.network,
				ln.address,
				false,
				ln.sockOptInts,
				ln.sockOptStrs,
			)
			ln.network = "udp"

		case "unix":
			_ = os.RemoveAll(ln.address)
			ln.fd, ln.addr, err = xsocket.UnixSocket(
				ln.network,
				ln.address,
				true,
				ln.sockOptInts,
				ln.sockOptStrs,
			)

		default:
			err = xerrors.ErrUnsupportedProtocol
		}
	})
	return
}

// close 关闭监听器。
func (ln *listener) close() {
	ln.closeOnce.Do(func() {
		if ln.fd > 0 {
			log.Error(os.NewSyscallError("close", unix.Close(ln.fd)))
		}

		ln.fd = -1

		if ln.network == "unix" {
			log.Error(os.RemoveAll(ln.address))
		}
	})
}

// initListener 初始化监听器。
func initListener(network, addr string, options *Options) (ln *listener, err error) {
	var (
		sockOptInts []xsocket.Option[int]
		sockOptStrs []xsocket.Option[string]
	)

	// ReusePort
	if options.ReusePort && network != "unix" {
		sockOptInts = append(sockOptInts, xsocket.Option[int]{
			SetSockOpt: xsocket.SetReuseport,
			Opt:        1,
		})
	}

	// ReuseAddr
	if options.ReuseAddr {
		sockOptInts = append(sockOptInts, xsocket.Option[int]{
			SetSockOpt: xsocket.SetReuseAddr,
			Opt:        1,
		})
	}

	// TCP_NODELAY
	if options.TCPNoDelay == TCPNoDelay && strings.HasPrefix(network, "tcp") {
		sockOptInts = append(sockOptInts, xsocket.Option[int]{
			SetSockOpt: xsocket.SetNoDelay,
			Opt:        1,
		})
	}

	// 接收缓冲区
	if options.SocketRecvBuffer > 0 {
		sockOptInts = append(sockOptInts, xsocket.Option[int]{
			SetSockOpt: xsocket.SetRecvBuffer,
			Opt:        options.SocketRecvBuffer,
		})
	}

	// 发送缓冲区
	if options.SocketSendBuffer > 0 {
		sockOptInts = append(sockOptInts, xsocket.Option[int]{
			SetSockOpt: xsocket.SetSendBuffer,
			Opt:        options.SocketSendBuffer,
		})
	}

	// UDP 组播
	if strings.HasPrefix(network, "udp") {
		udpAddr, resolveErr := net.ResolveUDPAddr(network, addr)
		if resolveErr == nil && udpAddr.IP.IsMulticast() {
			if sockoptFn := xsocket.SetMulticastMembership(network, udpAddr); sockoptFn != nil {
				sockOptInts = append(sockOptInts, xsocket.Option[int]{
					SetSockOpt: sockoptFn,
					Opt:        options.MulticastInterfaceIndex,
				})
			}
		}
	}

	// 绑定网卡
	if options.BindToDevice != "" {
		sockOptStrs = append(sockOptStrs, xsocket.Option[string]{
			SetSockOpt: xsocket.SetBindToDevice,
			Opt:        options.BindToDevice,
		})
	}

	ln = &listener{
		network:     network,
		address:     addr,
		sockOptInts: sockOptInts,
		sockOptStrs: sockOptStrs,
	}

	err = ln.open()

	// 在 Linux / FreeBSD / DragonFlyBSD 上，
	// TCP keepalive 选项会从监听套接字继承到已接受连接。
	if options.TCPKeepAlive > 0 &&
		ln.network == "tcp" &&
		(runtime.GOOS == "linux" || runtime.GOOS == "freebsd" || runtime.GOOS == "dragonfly") {
		err = setKeepAlive(
			ln.fd,
			true,
			options.TCPKeepAlive,
			options.TCPKeepInterval,
			options.TCPKeepCount,
		)
	}

	return
}
