package xants

import (
	"errors"
	"time"
)

// errQueueIsFull 表示 worker 队列已满。
var errQueueIsFull = errors.New("the queue is full")

// worker 定义 worker 的通用行为接口。
type worker interface {
	run()
	finish()
	lastUsedTime() int64
	setLastUsedTime(t int64)
	inputFunc(func())
	inputArg(any)
}

// workerQueue 定义 worker 队列接口。
type workerQueue interface {
	len() int
	isEmpty() bool
	insert(worker) error
	detach() worker
	refresh(duration time.Duration) []worker // 清理过期 worker，并返回这些 worker
	reset()
}

type queueType int

const (
	queueTypeStack queueType = 1 << iota
	queueTypeLoopQueue
)

// newWorkerQueue 根据队列类型创建对应的 worker 队列。
func newWorkerQueue(qType queueType, size int) workerQueue {
	switch qType {
	case queueTypeStack:
		return newWorkerStack(size)
	case queueTypeLoopQueue:
		return newWorkerLoopQueue(size)
	default:
		return newWorkerStack(size)
	}
}
