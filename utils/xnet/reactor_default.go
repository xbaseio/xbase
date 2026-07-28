//go:build (darwin || dragonfly || freebsd || linux || netbsd || openbsd) && !poll_opt

package xnet

import (
	"runtime"

	"github.com/xbaseio/xbase/utils/xnetpoll"
	"github.com/xbaseio/xbase/xerrors"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

// rotate 运行主 reactor，负责接收新连接。
func (el *eventloop) rotate() error {
	if el.engine.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	err := el.poller.Polling(el.accept0)
	if xerrors.Is(err, xerrors.ErrEngineShutdown) {
		xlog.Logger().Debug("main reactor is exiting in terms of the demand from user", zap.Error(err))
		err = nil
	} else if err != nil {
		xlog.Logger().Error("main reactor is exiting xbase to error", zap.Error(err))
	}

	el.engine.shutdown(err)
	return err
}

// orbit 运行子 reactor，只负责处理已建立连接的 IO 事件。
func (el *eventloop) orbit() error {
	if el.engine.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	err := el.poller.Polling(func(fd int, ev xnetpoll.IOEvent, flags xnetpoll.IOFlags) error {
		c := el.connections.getConn(fd)
		if c == nil {
			zap.S(
			// 对于 kqueue，这可能发生在连接已经关闭之后，
			// fd 会按照手册说明自动从 kqueue 中移除。
			//
			// 对于 epoll，它有时会为一个已经过期的 fd 继续投递事件，
			// 而这个 fd 已经不在我们的连接集合中。
			// 这里需要显式把它从 epoll 集合中删除，并记录警告日志。
			).Warnf(
				"received event[fd=%d|ev=%d|flags=%d] of a stale connection from event-loop(%d)",
				fd, ev, flags, el.idx,
			)
			return el.poller.Delete(fd)
		}

		return c.processIO(fd, ev, flags)
	})

	if xerrors.Is(err, xerrors.ErrEngineShutdown) {
		xlog.Logger().Debug("event-loop() is exiting in terms of the demand from user", zap.Any("idx", el.idx), zap.Error(err))
		err = nil
	} else if err != nil {
		xlog.Logger().Error("event-loop() is exiting xbase to error", zap.Any("idx", el.idx), zap.Error(err))
	}

	el.closeConns()
	el.engine.shutdown(err)

	return err
}

// run 运行单 reactor 模式的 event-loop，同时处理 accept 和连接 IO。
func (el *eventloop) run() error {
	if el.engine.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	err := el.poller.Polling(func(fd int, ev xnetpoll.IOEvent, flags xnetpoll.IOFlags) error {
		c := el.connections.getConn(fd)
		if c == nil {
			// 如果这是监听 fd，则处理 accept。
			if _, ok := el.listeners[fd]; ok {
				return el.accept(fd, ev, flags)
			}
			xlog.Logger().Warn("received event[fd=|ev=|flags=] of a stale connection from event-loop()", zap.Any("fd", fd), zap.Any("ev", ev), zap.Any("flags", flags), zap.Any("idx", el.idx))
			return el.poller.Delete(fd)
		}

		return c.processIO(fd, ev, flags)
	})

	if xerrors.Is(err, xerrors.ErrEngineShutdown) {
		xlog.Logger().Debug("event-loop() is exiting in terms of the demand from user", zap.Any("idx", el.idx), zap.Error(err))
		err = nil
	} else if err != nil {
		xlog.Logger().Error("event-loop() is exiting xbase to error", zap.Any("idx", el.idx), zap.Error(err))
	}

	el.closeConns()
	el.engine.shutdown(err)

	return err
}
