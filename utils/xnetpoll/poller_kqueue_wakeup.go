//go:build darwin || dragonfly || freebsd
// +build darwin dragonfly freebsd

package xnetpoll

import (
	"github.com/xbaseio/xbase/log"
	"golang.org/x/sys/unix"
)

func (p *Poller) addWakeupEvent() error {
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Flags:  unix.EV_ADD | unix.EV_CLEAR,
	}}, nil, nil)
	return err
}

func (p *Poller) wakePoller() error {
retry:
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{{
		Ident:  0,
		Filter: unix.EVFILT_USER,
		Fflags: unix.NOTE_TRIGGER,
	}}, nil, nil)
	if err == nil {
		return nil
	}
	if err == unix.EINTR {
		// 即使返回了 EINTR，changelist 中的所有变更理论上也应该已经生效。
		// 但这里仍然选择谨慎处理，再重试一次，确保 100% 提交成功。
		goto retry
	}
	log.Warnf("failed to wake up the poller: %v", err)
	return err
}

func (p *Poller) drainWakeupEvent() {}
