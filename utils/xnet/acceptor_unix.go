//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package xnet

import (
	"runtime"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xnetpoll"
	"github.com/xbaseio/xbase/utils/xqueue"
	"github.com/xbaseio/xbase/utils/xsocket"
	"github.com/xbaseio/xbase/xerrors"
)

// accept0 持续从监听套接字中批量接收连接，直到 accept 队列被取空。
func (el *eventloop) accept0(fd int, _ xnetpoll.IOEvent, _ xnetpoll.IOFlags) error {
	listener := el.listeners[fd]
	network := listener.network
	opts := el.engine.opts

	for {
		nfd, sa, err := xsocket.Accept(fd)
		switch err {
		case nil:
		case unix.EAGAIN:
			// accept 队列已经取空，当前批次处理完成
			return nil
		case unix.EINTR, unix.ECONNRESET, unix.ECONNABORTED:
			// ECONNRESET 或 ECONNABORTED 可能表示：
			// 连接在真正调用 Accept() 之前，就已经从 accept 队列中失效或被关闭。
			// 这类错误通常可以直接继续重试。
			continue
		default:
			log.Errorf("Accept() failed xbase to error: %v", err)
			return xerrors.ErrAcceptSocket
		}

		remoteAddr := xsocket.SockaddrToTCPOrUnixAddr(sa)

		if opts.TCPKeepAlive > 0 &&
			network == "tcp" &&
			(runtime.GOOS != "linux" && runtime.GOOS != "freebsd" && runtime.GOOS != "dragonfly") {
			// 在 Linux、FreeBSD、DragonFlyBSD 之外的平台上，
			// 已接受连接通常不会自动继承监听套接字上的 TCP KeepAlive 配置，
			// 因此这里需要对新连接显式设置。
			if err = setKeepAlive(
				nfd,
				true,
				opts.TCPKeepAlive,
				opts.TCPKeepInterval,
				opts.TCPKeepCount,
			); err != nil {
				log.Errorf("failed to set TCP keepalive on fd=%d: %v", fd, err)
			}
		}

		targetEL := el.engine.eventLoops.next(remoteAddr)
		c := newStreamConn(network, nfd, targetEL, sa, listener.addr, remoteAddr)

		if err = targetEL.poller.Trigger(xqueue.HighPriority, targetEL.register, c); err != nil {
			log.Errorf("failed to enqueue the accepted socket fd=%d to poller: %v", c.fd, err)
			_ = unix.Close(nfd)
			c.release()
		}
	}
}

// accept 处理单次接收连接事件。
// 如果是 UDP 监听，则转交给 readUDP 处理；否则接收一个 TCP/Unix 连接并注册到当前事件循环。
func (el *eventloop) accept(fd int, ev xnetpoll.IOEvent, flags xnetpoll.IOFlags) error {
	listener := el.listeners[fd]
	network := listener.network
	opts := el.engine.opts

	if network == "udp" {
		return el.readUDP(fd, ev, flags)
	}

	nfd, sa, err := xsocket.Accept(fd)
	switch err {
	case nil:
	case unix.EINTR, unix.EAGAIN, unix.ECONNRESET, unix.ECONNABORTED:
		// ECONNRESET 或 ECONNABORTED 可能表示：
		// accept 队列中的连接在调用 Accept() 之前已经关闭。
		// 这类错误通常无需中断事件循环，直接返回等待下一次事件即可。
		return nil
	default:
		log.Errorf("Accept() failed xbase to error: %v", err)
		return xerrors.ErrAcceptSocket
	}

	remoteAddr := xsocket.SockaddrToTCPOrUnixAddr(sa)

	if opts.TCPKeepAlive > 0 &&
		network == "tcp" &&
		(runtime.GOOS != "linux" && runtime.GOOS != "freebsd" && runtime.GOOS != "dragonfly") {
		// 在 Linux、FreeBSD、DragonFlyBSD 之外的平台上，
		// 已接受连接不会继承监听套接字上的 TCP KeepAlive 配置，
		// 因此需要在这里显式设置。
		if err = setKeepAlive(
			nfd,
			true,
			opts.TCPKeepAlive,
			opts.TCPKeepInterval,
			opts.TCPKeepCount,
		); err != nil {
			log.Errorf("failed to set TCP keepalive on fd=%d: %v", fd, err)
		}
	}

	c := newStreamConn(network, nfd, el, sa, listener.addr, remoteAddr)
	return el.register0(c)
}
