package xnet

import (
	"context"
	"net"
	"os"
	"sync"
	"syscall"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/xerrors"
	"golang.org/x/sys/windows"
)

// listener 表示一个监听器。
type listener struct {
	openOnce, closeOnce sync.Once

	network string
	address string

	lc   *net.ListenConfig
	ln   net.Listener
	pc   net.PacketConn
	addr net.Addr
}

// dup 复制监听器底层 handle。
func (l *listener) dup() (int, error) {
	if l.ln == nil && l.pc == nil {
		return -1, xerrors.ErrUnsupportedOp
	}

	var (
		sc syscall.Conn
		ok bool
	)

	if l.ln != nil {
		sc, ok = l.ln.(syscall.Conn)
	} else {
		sc, ok = l.pc.(syscall.Conn)
	}

	if !ok {
		return -1, xerrors.New("failed to convert net.Conn to syscall.Conn")
	}

	rc, err := sc.SyscallConn()
	if err != nil {
		return -1, xerrors.New("failed to get syscall.RawConn from net.Conn")
	}

	var dupHandle windows.Handle
	e := rc.Control(func(fd uintptr) {
		process := windows.CurrentProcess()
		err = windows.DuplicateHandle(
			process,
			windows.Handle(fd),
			process,
			&dupHandle,
			0,
			true,
			windows.DUPLICATE_SAME_ACCESS,
		)
	})
	if err != nil {
		return -1, err
	}
	if e != nil {
		return -1, e
	}

	return int(dupHandle), nil
}

// open 打开监听器。
func (l *listener) open() (err error) {
	l.openOnce.Do(func() {
		switch l.network {
		case "udp", "udp4", "udp6":
			if l.pc, err = l.lc.ListenPacket(context.Background(), l.network, l.address); err == nil {
				l.addr = l.pc.LocalAddr()
			}

		case "unix":
			_ = os.Remove(l.address)
			fallthrough

		case "tcp", "tcp4", "tcp6":
			if l.ln, err = l.lc.Listen(context.Background(), l.network, l.address); err == nil {
				l.addr = l.ln.Addr()
			}

		default:
			err = xerrors.ErrUnsupportedProtocol
		}
	})
	return
}

// close 关闭监听器。
func (l *listener) close() {
	l.closeOnce.Do(func() {
		if l.pc != nil {
			log.Error(os.NewSyscallError("close", l.pc.Close()))
			return
		}

		l.pc = nil
		log.Error(os.NewSyscallError("close", l.ln.Close()))
	})
}

// initListener 初始化监听器。
func initListener(network, addr string, options *Options) (*listener, error) {
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				if network != "unix" && (options.ReuseAddr || options.ReusePort) {
					_ = windows.SetsockoptInt(
						windows.Handle(fd),
						windows.SOL_SOCKET,
						windows.SO_REUSEADDR,
						1,
					)
				}

				if options.TCPNoDelay == TCPNoDelay {
					_ = windows.SetsockoptInt(
						windows.Handle(fd),
						windows.IPPROTO_TCP,
						windows.TCP_NODELAY,
						1,
					)
				}

				if options.SocketRecvBuffer > 0 {
					_ = windows.SetsockoptInt(
						windows.Handle(fd),
						windows.SOL_SOCKET,
						windows.SO_RCVBUF,
						options.SocketRecvBuffer,
					)
				}

				if options.SocketSendBuffer > 0 {
					_ = windows.SetsockoptInt(
						windows.Handle(fd),
						windows.SOL_SOCKET,
						windows.SO_SNDBUF,
						options.SocketSendBuffer,
					)
				}
			})
		},
		KeepAlive: options.TCPKeepAlive,
	}

	l := listener{
		network: network,
		address: addr,
		lc:      &lc,
	}

	return &l, l.open()
}
