//go:build netbsd || openbsd
// +build netbsd openbsd

package xnetpoll

import (
	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/log"
)

func (p *Poller) addWakeupEvent() error {
	// 创建 pipe（用于唤醒 poller）
	p.pipe = make([]int, 2)
	if err := unix.Pipe2(p.pipe[:], unix.O_NONBLOCK|unix.O_CLOEXEC); err != nil {
		log.Fatalf("failed to create pipe for wakeup event: %v", err)
	}

	// 将 pipe 的读端注册到 kqueue，监听读事件
	_, err := unix.Kevent(p.fd, []unix.Kevent_t{{
		Ident:  uint32(p.pipe[0]),
		Filter: unix.EVFILT_READ,
		Flags:  unix.EV_ADD,
	}}, nil, nil)

	return err
}

func (p *Poller) wakePoller() error {
retry:
	// 向 pipe 写入数据以唤醒 poller
	_, err := unix.Write(p.pipe[1], []byte("x"))
	if err == nil || err == unix.EAGAIN {
		return nil
	}
	if err == unix.EINTR {
		// 被信号中断，重试
		goto retry
	}

	log.Warnf("failed to write to the wakeup pipe: %v", err)
	return err
}

func (p *Poller) drainWakeupEvent() {
	// 读取 pipe 数据，清空唤醒信号
	var buf [8]byte
	_, _ = unix.Read(p.pipe[0], buf[:])
}
