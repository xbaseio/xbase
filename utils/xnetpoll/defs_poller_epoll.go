//go:build linux
// +build linux

package xnetpoll

import "golang.org/x/sys/unix"

// IOFlags 表示 I/O 事件的标志位类型。
type IOFlags = uint16

// IOEvent 表示 I/O 事件类型（Linux）。
type IOEvent = uint32

const (
	// InitPollEventsCap 初始事件列表容量。
	InitPollEventsCap = 128

	// MaxPollEventsCap 事件列表最大容量。
	MaxPollEventsCap = 1024

	// MinPollEventsCap 事件列表最小容量。
	MinPollEventsCap = 32

	// MaxAsyncTasksAtOneTime 单次循环最多处理的异步任务数。
	MaxAsyncTasksAtOneTime = 256

	// ReadEvents 表示可读事件（epoll）。
	ReadEvents = unix.EPOLLIN | unix.EPOLLPRI

	// WriteEvents 表示可写事件（epoll）。
	WriteEvents = unix.EPOLLOUT

	// ReadWriteEvents 表示可读写事件。
	ReadWriteEvents = ReadEvents | WriteEvents

	// ErrEvents 表示错误事件。
	ErrEvents = unix.EPOLLERR | unix.EPOLLHUP
)

// IsReadEvent 判断是否为读事件。
func IsReadEvent(event IOEvent) bool {
	return event&ReadEvents != 0
}

// IsWriteEvent 判断是否为写事件。
func IsWriteEvent(event IOEvent) bool {
	return event&WriteEvents != 0
}

// IsErrorEvent 判断是否为错误事件。
func IsErrorEvent(event IOEvent, _ IOFlags) bool {
	return event&ErrEvents != 0
}

type eventList struct {
	size   int
	events []epollevent
}

func newEventList(size int) *eventList {
	return &eventList{
		size:   size,
		events: make([]epollevent, size),
	}
}

func (el *eventList) expand() {
	if newSize := el.size << 1; newSize <= MaxPollEventsCap {
		el.size = newSize
		el.events = make([]epollevent, newSize)
	}
}

func (el *eventList) shrink() {
	if newSize := el.size >> 1; newSize >= MinPollEventsCap {
		el.size = newSize
		el.events = make([]epollevent, newSize)
	}
}
