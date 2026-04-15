package xants

// PoolWithFuncGeneric 是 PoolWithFunc 的泛型版本。
type PoolWithFuncGeneric[T any] struct {
	*poolCommon

	// fn 是统一的任务处理函数。
	fn func(T)
}

// Invoke 将参数提交到协程池并启动一个新任务。
func (p *PoolWithFuncGeneric[T]) Invoke(arg T) error {
	if p.IsClosed() {
		return ErrPoolClosed
	}

	w, err := p.retrieveWorker()
	if w != nil {
		w.(*goWorkerWithFuncGeneric[T]).arg <- arg
	}
	return err
}

// NewPoolWithFuncGeneric 使用自定义配置创建一个 PoolWithFuncGeneric[T]。
func NewPoolWithFuncGeneric[T any](size int, pf func(T), options ...Option) (*PoolWithFuncGeneric[T], error) {
	if pf == nil {
		return nil, ErrLackPoolFunc
	}

	pc, err := newPool(size, options...)
	if err != nil {
		return nil, err
	}

	pool := &PoolWithFuncGeneric[T]{
		poolCommon: pc,
		fn:         pf,
	}

	pool.workerCache.New = func() any {
		return &goWorkerWithFuncGeneric[T]{
			pool: pool,
			arg:  make(chan T, workerChanCap),
			exit: make(chan struct{}, 1),
		}
	}

	return pool, nil
}
