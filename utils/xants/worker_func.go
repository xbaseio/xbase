package xants

import (
	"runtime/debug"
)

// goWorkerWithFunc 是真正执行任务的 worker。
// 它会启动一个 goroutine，接收参数并调用统一函数执行任务。
type goWorkerWithFunc struct {
	worker

	// pool 表示拥有该 worker 的协程池。
	pool *PoolWithFunc

	// arg 表示传给任务函数的参数。
	arg chan any

	// lastUsed 表示该 worker 最近一次被使用的时间。
	lastUsed int64
}

// run 启动一个 goroutine，循环执行任务处理流程。
func (w *goWorkerWithFunc) run() {
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

		for arg := range w.arg {
			if arg == nil {
				return
			}
			w.pool.fn(arg)
			if ok := w.pool.revertWorker(w); !ok {
				return
			}
		}
	}()
}

func (w *goWorkerWithFunc) finish() {
	w.arg <- nil
}

func (w *goWorkerWithFunc) lastUsedTime() int64 {
	return w.lastUsed
}

func (w *goWorkerWithFunc) setLastUsedTime(t int64) {
	w.lastUsed = t
}

func (w *goWorkerWithFunc) inputArg(arg any) {
	w.arg <- arg
}
