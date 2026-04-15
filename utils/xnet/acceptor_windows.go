package xnet

import (
	"errors"
	"net"
	"runtime"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xpool/xgoroutine"
	"github.com/xbaseio/xbase/xerrors"
)

func (eng *engine) listenStream(ln net.Listener) (err error) {
	if eng.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	defer func() { eng.shutdown(err) }()

	for {
		// Accept TCP socket.
		tc, e := ln.Accept()
		if e != nil {
			err = e
			if !eng.beingShutdown.Load() {
				log.Errorf("Accept() fails xbase to error: %v", err)
			} else if errors.Is(err, net.ErrClosed) {
				err = errors.Join(err, xerrors.ErrEngineShutdown)
			}
			return
		}
		el := eng.eventLoops.next(tc.RemoteAddr())
		c := newStreamConn(el, tc, nil)
		el.ch <- &openConn{c: c}
		xgoroutine.DefaultWorkerPool.Submit(func() {
			var buffer [0x10000]byte
			for {
				n, err := tc.Read(buffer[:])
				if err != nil {
					el.ch <- &netErr{c, err}
					return
				}
				el.ch <- packTCPConn(c, buffer[:n])
			}
		})
	}
}

func (eng *engine) ListenUDP(pc net.PacketConn) (err error) {
	if eng.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	defer func() { eng.shutdown(err) }()

	var buffer [0x10000]byte
	for {
		// Read data from UDP socket.
		n, addr, e := pc.ReadFrom(buffer[:])
		if e != nil {
			err = e
			if !eng.beingShutdown.Load() {
				log.Errorf("failed to receive data from UDP fd xbase to error:%v", err)
			} else if errors.Is(err, net.ErrClosed) {
				err = errors.Join(err, xerrors.ErrEngineShutdown)
			}
			return
		}
		el := eng.eventLoops.next(addr)
		c := newUDPConn(el, pc, nil, pc.LocalAddr(), addr, nil)
		el.ch <- packUDPConn(c, buffer[:n])
	}
}
