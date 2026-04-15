//go:build (darwin || dragonfly || freebsd || linux || netbsd || openbsd) && poll_opt

package xnet

import (
	"runtime"

	"github.com/xbaseio/xbase/xerrors"
)

// rotate 运行主 reactor，负责接收连接。
func (el *eventloop) rotate() error {
	if el.engine.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	err := el.poller.Polling()
	if xerrors.Is(err, xerrors.ErrEngineShutdown) {
		el.getLogger().Debugf("main reactor is exiting in terms of the demand from user, %v", err)
		err = nil
	} else if err != nil {
		el.getLogger().Errorf("main reactor is exiting xbase to error: %v", err)
	}

	el.engine.shutdown(err)
	return err
}

// orbit 运行子 reactor，只处理已建立连接的 IO 事件。
func (el *eventloop) orbit() error {
	if el.engine.opts.LockOSThread {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
	}

	err := el.poller.Polling()
	if xerrors.Is(err, xerrors.ErrEngineShutdown) {
		el.getLogger().Debugf("event-loop(%d) is exiting in terms of the demand from user, %v", el.idx, err)
		err = nil
	} else if err != nil {
		el.getLogger().Errorf("event-loop(%d) is exiting xbase to error: %v", el.idx, err)
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

	err := el.poller.Polling()
	if xerrors.Is(err, xerrors.ErrEngineShutdown) {
		el.getLogger().Debugf("event-loop(%d) is exiting in terms of the demand from user, %v", el.idx, err)
		err = nil
	} else if err != nil {
		el.getLogger().Errorf("event-loop(%d) is exiting xbase to error: %v", el.idx, err)
	}

	el.closeConns()
	el.engine.shutdown(err)

	return err
}
