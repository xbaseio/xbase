// Package xqueue 实现了一个用于异步任务的无锁队列。
package xqueue

import "sync"

// Func 表示由 poller 执行的回调函数。
type Func func(any) error

// Task 是一个任务封装，包含执行函数及其参数。
type Task struct {
	Exec  Func // 执行函数
	Param any  // 参数
}

// taskPool 用于复用 Task 对象，减少 GC 压力。
var taskPool = sync.Pool{New: func() any { return new(Task) }}

// GetTask 从对象池中获取一个 Task（可能是复用的）。
func GetTask() *Task {
	return taskPool.Get().(*Task)
}

// PutTask 将已使用的 Task 放回对象池（清理字段防止内存泄漏）。
func PutTask(task *Task) {
	task.Exec, task.Param = nil, nil
	taskPool.Put(task)
}

// AsyncTaskQueue 定义了一个异步任务队列接口。
type AsyncTaskQueue interface {
	Enqueue(*Task)  // 入队
	Dequeue() *Task // 出队（无任务返回 nil）
	IsEmpty() bool  // 是否为空
	Length() int32  // 队列长度（近似值）
}

// EventPriority 表示任务优先级。
type EventPriority int

const (
	// HighPriority 表示需要尽快执行的任务（高优先级）。
	HighPriority EventPriority = iota

	// LowPriority 表示可以稍微延迟执行的任务（低优先级）。
	LowPriority
)
