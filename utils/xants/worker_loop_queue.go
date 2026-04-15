package xants

import "time"

// loopQueue 是一个环形 worker 队列。
// 用于存放空闲 worker，并支持按 lastUsed 时间批量回收过期 worker。
type loopQueue struct {
	items  []worker // 环形队列存储
	expiry []worker // 临时保存本次回收的过期 worker
	head   int      // 队头
	tail   int      // 队尾（下一个写入位置）
	size   int      // 队列容量
	isFull bool     // 队列是否已满
}

// newWorkerLoopQueue 创建一个固定容量的环形 worker 队列。
func newWorkerLoopQueue(size int) *loopQueue {
	if size <= 0 {
		return nil
	}
	return &loopQueue{
		items: make([]worker, size),
		size:  size,
	}
}

// len 返回当前队列中的元素数量。
func (wq *loopQueue) len() int {
	if wq.size == 0 || wq.isEmpty() {
		return 0
	}

	if wq.head == wq.tail && wq.isFull {
		return wq.size
	}

	if wq.tail > wq.head {
		return wq.tail - wq.head
	}

	return wq.size - wq.head + wq.tail
}

// isEmpty 返回队列是否为空。
func (wq *loopQueue) isEmpty() bool {
	return wq.head == wq.tail && !wq.isFull
}

// insert 向队尾插入一个 worker。
// 如果队列已满，返回 errQueueIsFull。
func (wq *loopQueue) insert(w worker) error {
	if wq.isFull {
		return errQueueIsFull
	}

	wq.items[wq.tail] = w
	wq.tail = (wq.tail + 1) % wq.size

	if wq.tail == wq.head {
		wq.isFull = true
	}

	return nil
}

// detach 从队头取出一个 worker。
// 如果队列为空，返回 nil。
func (wq *loopQueue) detach() worker {
	if wq.isEmpty() {
		return nil
	}

	w := wq.items[wq.head]
	wq.items[wq.head] = nil
	wq.head = (wq.head + 1) % wq.size
	wq.isFull = false

	return w
}

// refresh 回收超过指定时长未使用的 worker，并返回这些过期 worker。
func (wq *loopQueue) refresh(duration time.Duration) []worker {
	expiryTime := time.Now().Add(-duration).UnixNano()
	index := wq.binarySearch(expiryTime)
	if index == -1 {
		return nil
	}

	wq.expiry = wq.expiry[:0]

	if wq.head <= index {
		wq.expiry = append(wq.expiry, wq.items[wq.head:index+1]...)
		for i := wq.head; i < index+1; i++ {
			wq.items[i] = nil
		}
	} else {
		wq.expiry = append(wq.expiry, wq.items[0:index+1]...)
		wq.expiry = append(wq.expiry, wq.items[wq.head:]...)
		for i := 0; i < index+1; i++ {
			wq.items[i] = nil
		}
		for i := wq.head; i < wq.size; i++ {
			wq.items[i] = nil
		}
	}

	head := (index + 1) % wq.size
	wq.head = head
	if len(wq.expiry) > 0 {
		wq.isFull = false
	}

	return wq.expiry
}

// binarySearch 在环形队列中查找最后一个“已过期”的 worker 下标。
// 如果不存在过期 worker，则返回 -1。
func (wq *loopQueue) binarySearch(expiryTime int64) int {
	var mid, nlen, basel, tmid int
	nlen = len(wq.items)

	// 如果队列为空，或队头元素都未过期，则无需回收。
	if wq.isEmpty() || expiryTime < wq.items[wq.head].lastUsedTime() {
		return -1
	}

	// 示例：
	// size = 8, head = 7, tail = 4
	// [ 2, 3, 4, 5, nil, nil, nil, 1 ]   实际存储位置
	//   0  1  2  3    4   5    6    7
	//              tail             head
	//
	// 映射后可视为：
	// [ 1, 2, 3, 4, nil, nil, nil, 0 ]   逻辑位置
	//   l                           r

	// 这里的二分思路来自 worker_stack：
	// 先把环形队列映射成“逻辑连续区间”，再在映射区间中做二分查找。
	r := (wq.tail - 1 - wq.head + nlen) % nlen
	basel = wq.head
	l := 0

	for l <= r {
		mid = l + ((r - l) >> 1) // 避免 mid 计算溢出

		// 将映射下标转换回真实下标
		tmid = (mid + basel + nlen) % nlen

		if expiryTime < wq.items[tmid].lastUsedTime() {
			r = mid - 1
		} else {
			l = mid + 1
		}
	}

	// 返回真实下标
	return (r + basel + nlen) % nlen
}

// reset 清空整个队列，并结束其中所有 worker。
func (wq *loopQueue) reset() {
	for w := wq.detach(); w != nil; w = wq.detach() {
		w.finish()
	}

	wq.head = 0
	wq.tail = 0
}
