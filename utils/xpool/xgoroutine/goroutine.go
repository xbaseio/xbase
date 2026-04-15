package xgoroutine

import (
	"time"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xants"
)

const (
	// DefaultAntsPoolSize sets up the capacity of worker pool, 256 * 1024.
	DefaultAntsPoolSize = 1 << 18

	// ExpiryDuration is the interval time to clean up those expired workers.
	ExpiryDuration = 10 * time.Second

	// Nonblocking decides what to do when submitting a new task to a full worker pool: waiting for a available worker
	// or returning nil directly.
	Nonblocking = true
)

func init() {
	// It releases the default pool from xants.
	xants.Release()
}

// DefaultWorkerPool is the global worker pool.
var DefaultWorkerPool = Default()

// Pool is the alias of xants.Pool.
type Pool = xants.Pool

type antsLogger struct {
	log.Logger
}

// Printf implements the xants.Logger interface.
func (l antsLogger) Printf(format string, args ...any) {
	l.Infof(format, args...)
}

// Default instantiates a non-blocking goroutine pool with the capacity of DefaultAntsPoolSize.
func Default() *Pool {
	options := xants.Options{
		ExpiryDuration: ExpiryDuration,
		Nonblocking:    Nonblocking,
		Logger:         &antsLogger{log.GetLogger()},
		PanicHandler: func(a any) {
			log.Errorf("goroutine pool panic: %v", a)
		},
	}
	defaultAntsPool, _ := xants.NewPool(DefaultAntsPoolSize, xants.WithOptions(options))
	return defaultAntsPool
}
