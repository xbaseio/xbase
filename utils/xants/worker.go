package xants

import (
	"runtime/debug"
)

// goWorker 是真正执行任务的 worker。
// 它会启动一个 goroutine，接收任务并执行函数调用。
type goWorker struct {
	worker

	// pool 表示拥有该 worker 的协程池。
	pool *Pool

	// task 表示待执行任务。
	task chan func()

	// lastUsed 表示该 worker 最近一次被使用的时间。
	lastUsed int64
}

// run 启动一个 goroutine，循环执行任务处理流程。
func (w *goWorker) run() {
	w.pool.addRunning(1)
	go func() {
		defer func() {
			if w.pool.addRunning(-1) == 0 && w.pool.IsClosed() {
				w.pool.once.Do(func() {
					close(w.pool.allDone)
				})
			}
			w.pool.workerCache.Put(w)
			if p := recover(); p != nil {
				if ph := w.pool.options.PanicHandler; ph != nil {
					ph(p)
				} else {
					w.pool.options.Logger.Printf("worker exits from panic: %v\n%s\n", p, debug.Stack())
				}
			}
			// 在这里调用 Signal()，避免有协程一直等待可用 worker。
			w.pool.cond.Signal()
		}()

		for fn := range w.task {
			if fn == nil {
				return
			}
			fn()
			if ok := w.pool.revertWorker(w); !ok {
				return
			}
		}
	}()
}

func (w *goWorker) finish() {
	w.task <- nil
}

func (w *goWorker) lastUsedTime() int64 {
	return w.lastUsed
}

func (w *goWorker) setLastUsedTime(t int64) {
	w.lastUsed = t
}

func (w *goWorker) inputFunc(fn func()) {
	w.task <- fn
}
