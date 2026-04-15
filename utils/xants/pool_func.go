package xants

// PoolWithFunc 类似 Pool，但所有 worker 执行同一个函数。
type PoolWithFunc struct {
	*poolCommon

	// fn 是统一的任务处理函数。
	fn func(any)
}

// Invoke 向协程池提交参数并执行任务。
//
// 注意：
// 允许在 Pool.Invoke() 内再次调用 Pool.Invoke()（递归提交任务），
// 但需要特别注意：
// 当池容量耗尽时，最后一次 Invoke 会发生阻塞。
// 若要避免阻塞，应在创建池时使用 xants.WithNonblocking(true)。
func (p *PoolWithFunc) Invoke(arg any) error {
	if p.IsClosed() {
		return ErrPoolClosed
	}

	w, err := p.retrieveWorker()
	if w != nil {
		w.inputArg(arg)
	}
	return err
}

// NewPoolWithFunc 使用自定义配置创建一个 PoolWithFunc。
func NewPoolWithFunc(size int, pf func(any), options ...Option) (*PoolWithFunc, error) {
	if pf == nil {
		return nil, ErrLackPoolFunc
	}

	pc, err := newPool(size, options...)
	if err != nil {
		return nil, err
	}

	pool := &PoolWithFunc{
		poolCommon: pc,
		fn:         pf,
	}

	pool.workerCache.New = func() any {
		return &goWorkerWithFunc{
			pool: pool,
			arg:  make(chan any, workerChanCap),
		}
	}

	return pool, nil
}
