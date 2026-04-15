package xants

import "time"

// workerStack 是基于切片实现的 worker 栈。
// 空闲 worker 按栈结构存放，并支持按最后使用时间回收过期 worker。
type workerStack struct {
	items  []worker // worker 列表
	expiry []worker // 临时保存本次回收的过期 worker
}

// newWorkerStack 创建一个 worker 栈。
func newWorkerStack(size int) *workerStack {
	return &workerStack{
		items: make([]worker, 0, size),
	}
}

// len 返回当前栈中 worker 数量。
func (ws *workerStack) len() int {
	return len(ws.items)
}

// isEmpty 返回栈是否为空。
func (ws *workerStack) isEmpty() bool {
	return len(ws.items) == 0
}

// insert 向栈中压入一个 worker。
func (ws *workerStack) insert(w worker) error {
	ws.items = append(ws.items, w)
	return nil
}

// detach 从栈顶弹出一个 worker。
// 如果栈为空，返回 nil。
func (ws *workerStack) detach() worker {
	l := ws.len()
	if l == 0 {
		return nil
	}

	w := ws.items[l-1]
	ws.items[l-1] = nil // 避免内存泄漏
	ws.items = ws.items[:l-1]

	return w
}

// refresh 回收超过指定时长未使用的 worker，并返回这些过期 worker。
func (ws *workerStack) refresh(duration time.Duration) []worker {
	n := ws.len()
	if n == 0 {
		return nil
	}

	expiryTime := time.Now().Add(-duration).UnixNano()
	index := ws.binarySearch(0, n-1, expiryTime)

	ws.expiry = ws.expiry[:0]
	if index != -1 {
		ws.expiry = append(ws.expiry, ws.items[:index+1]...)

		m := copy(ws.items, ws.items[index+1:])
		for i := m; i < n; i++ {
			ws.items[i] = nil
		}
		ws.items = ws.items[:m]
	}

	return ws.expiry
}

// binarySearch 在 worker 栈中查找最后一个“已过期”的 worker 下标。
// 如果不存在过期 worker，则返回 -1。
func (ws *workerStack) binarySearch(l, r int, expiryTime int64) int {
	for l <= r {
		mid := l + ((r - l) >> 1) // 避免 mid 计算溢出
		if expiryTime < ws.items[mid].lastUsedTime() {
			r = mid - 1
		} else {
			l = mid + 1
		}
	}
	return r
}

// reset 清空整个栈，并结束其中所有 worker。
func (ws *workerStack) reset() {
	for i := 0; i < ws.len(); i++ {
		ws.items[i].finish()
		ws.items[i] = nil
	}
	ws.items = ws.items[:0]
}
