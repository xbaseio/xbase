// Package queue 提供一个基于无锁（lock-free）并发队列的实现，
// 该算法来源于 Maged M. Michael 和 Michael L. Scott 在 1996 年提出的论文：
// https://dl.acm.org/doi/10.1145/248052.248106
//
// 非阻塞并发队列算法伪代码（简化说明）：
/*
	structure pointer_t {ptr: 指向 node_t 的指针, count: 计数器}
	structure node_t {value: 数据, next: pointer_t}
	structure queue_t {Head: pointer_t, Tail: pointer_t}

	initialize(Q)
	node = new_node()          // 创建一个空节点（哨兵节点）
	node->next.ptr = NULL
	Q->Head.ptr = Q->Tail.ptr = node

	enqueue(Q, value)
	// 核心思想：CAS + 自旋，把新节点挂到 tail 后面

	dequeue(Q)
	// 核心思想：CAS 移动 head，取 next.value
*/

package xqueue

import (
	"sync/atomic"
	"unsafe"
)

// lockFreeQueue 是一个简单、高性能、实用的无锁并发队列实现。
type lockFreeQueue struct {
	head   unsafe.Pointer // 指向头节点（哨兵节点）
	tail   unsafe.Pointer // 指向尾节点
	length int32          // 队列长度（近似值）
}

// node 表示队列中的一个节点。
type node struct {
	value *Task          // 存储任务
	next  unsafe.Pointer // 指向下一个节点
}

// NewLockFreeQueue 创建并返回一个新的无锁队列。
func NewLockFreeQueue() AsyncTaskQueue {
	n := unsafe.Pointer(&node{}) // 初始化哨兵节点（dummy node）
	return &lockFreeQueue{head: n, tail: n}
}

// Enqueue 将任务加入队列尾部。
func (q *lockFreeQueue) Enqueue(task *Task) {
	n := &node{value: task}

retry:
	tail := load(&q.tail)
	next := load(&tail.next)

	// 判断 tail 是否一致（防止并发修改）
	if tail == load(&q.tail) {

		if next == nil {
			// tail 确实是最后一个节点，尝试插入新节点
			if cas(&tail.next, next, n) {
				// 插入成功，尝试推进 tail 指针
				cas(&q.tail, tail, n)
				atomic.AddInt32(&q.length, 1)
				return
			}
		} else {
			// tail 落后了，帮忙推进 tail
			cas(&q.tail, tail, next)
		}
	}

	// 自旋重试
	goto retry
}

// Dequeue 从队列头部取出一个任务。
// 如果队列为空，返回 nil。
func (q *lockFreeQueue) Dequeue() *Task {
retry:
	head := load(&q.head)
	tail := load(&q.tail)
	next := load(&head.next)

	// 判断 head 是否一致
	if head == load(&q.head) {

		// 队列为空 或 tail 落后
		if head == tail {

			// 队列为空
			if next == nil {
				return nil
			}

			// tail 落后，尝试推进
			cas(&q.tail, tail, next)

		} else {
			// 先读取值（防止被释放）
			task := next.value

			// 尝试移动 head 指针
			if cas(&q.head, head, next) {
				atomic.AddInt32(&q.length, -1)
				return task
			}
		}
	}

	// 自旋重试
	goto retry
}

// IsEmpty 判断队列是否为空。
func (q *lockFreeQueue) IsEmpty() bool {
	return atomic.LoadInt32(&q.length) == 0
}

// Length 返回队列长度（注意：并发下是近似值）。
func (q *lockFreeQueue) Length() int32 {
	return atomic.LoadInt32(&q.length)
}

// load 原子读取指针并转换为 node。
func load(p *unsafe.Pointer) (n *node) {
	return (*node)(atomic.LoadPointer(p))
}

// cas 原子比较并交换（Compare-And-Swap）。
func cas(p *unsafe.Pointer, old, new *node) bool { //nolint:revive
	return atomic.CompareAndSwapPointer(p, unsafe.Pointer(old), unsafe.Pointer(new))
}
