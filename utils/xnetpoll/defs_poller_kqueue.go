//go:build darwin || dragonfly || freebsd || netbsd || openbsd
// +build darwin dragonfly freebsd netbsd openbsd

package xnetpoll

import "golang.org/x/sys/unix"

const (
	// InitPollEventsCap 初始事件列表容量。
	InitPollEventsCap = 64

	// MaxPollEventsCap 事件列表最大容量。
	MaxPollEventsCap = 512

	// MinPollEventsCap 事件列表最小容量。
	MinPollEventsCap = 16

	// MaxAsyncTasksAtOneTime 单次循环最多处理的异步任务数。
	MaxAsyncTasksAtOneTime = 128

	// ReadEvents 表示可读事件（kqueue）。
	ReadEvents = unix.EVFILT_READ

	// WriteEvents 表示可写事件（kqueue）。
	WriteEvents = unix.EVFILT_WRITE

	// ReadWriteEvents 表示可读写事件。
	ReadWriteEvents = ReadEvents | WriteEvents

	// ErrEvents 表示错误事件。
	ErrEvents = unix.EV_EOF | unix.EV_ERROR
)

// IsReadEvent 判断是否为读事件。
func IsReadEvent(event IOEvent) bool {
	return event == ReadEvents
}

// IsWriteEvent 判断是否为写事件。
func IsWriteEvent(event IOEvent) bool {
	return event == WriteEvents
}

// IsErrorEvent 判断是否为错误事件。
func IsErrorEvent(_ IOEvent, flags IOFlags) bool {
	return flags&ErrEvents != 0
}

type eventList struct {
	size   int
	events []unix.Kevent_t
}

func newEventList(size int) *eventList {
	return &eventList{
		size:   size,
		events: make([]unix.Kevent_t, size),
	}
}

func (el *eventList) expand() {
	if newSize := el.size << 1; newSize <= MaxPollEventsCap {
		el.size = newSize
		el.events = make([]unix.Kevent_t, newSize)
	}
}

func (el *eventList) shrink() {
	if newSize := el.size >> 1; newSize >= MinPollEventsCap {
		el.size = newSize
		el.events = make([]unix.Kevent_t, newSize)
	}
}
