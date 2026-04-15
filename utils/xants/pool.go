package xants

// Pool 是一个协程池，用于限制和复用大量 goroutine。
// 池容量可以是固定的，也可以是无限的。
type Pool struct {
	*poolCommon
}

// Submit 向协程池提交任务。
//
// 注意：
// 允许在 Pool.Submit() 内再次调用 Pool.Submit()（递归提交任务），
// 但需要特别注意：
// 当池容量耗尽时，最后一次 Submit 会发生阻塞。
// 若要避免阻塞，应在创建池时使用 xants.WithNonblocking(true)。
func (p *Pool) Submit(task func()) error {
	if p.IsClosed() {
		return ErrPoolClosed
	}

	w, err := p.retrieveWorker()
	if w != nil {
		w.inputFunc(task)
	}
	return err
}

// NewPool 使用自定义配置创建一个 Pool。
func NewPool(size int, options ...Option) (*Pool, error) {
	pc, err := newPool(size, options...)
	if err != nil {
		return nil, err
	}

	pool := &Pool{poolCommon: pc}

	pool.workerCache.New = func() any {
		return &goWorker{
			pool: pool,
			task: make(chan func(), workerChanCap),
		}
	}

	return pool, nil
}
