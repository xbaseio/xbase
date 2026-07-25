package versionretire

import (
	"context"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/xlog"
)

type Options struct {
	Ctx         context.Context
	Registry    registry.Registry
	ServiceName string
	Kind        string
	ID          string
	Version     string
	Alias       string
	GameID      int32
	MatchGameID bool
	RetireDelay time.Duration
	Timeout     time.Duration
	Shutdown    func()
}

// Start 监听同组服务版本，低版本在等待期后执行 Shutdown
func Start(opts Options) {
	if opts.Registry == nil || opts.Shutdown == nil {
		return
	}

	if opts.Timeout <= 0 {
		opts.Timeout = 3 * time.Second
	}

	go func() {
		w := &watcher{opts: opts}

		ctx, cancel := context.WithTimeout(opts.Ctx, opts.Timeout)
		watcher, err := opts.Registry.Watch(ctx, opts.ServiceName)
		cancel()
		if err != nil {
			xlog.Sugar().Fatalf("%s version watch failed: %v", opts.Kind, err)
		}

		defer watcher.Stop()

		tctx, tcancel := context.WithTimeout(opts.Ctx, opts.Timeout)
		if services, err := opts.Registry.Services(tctx, opts.ServiceName); err == nil {
			w.check(services)
		}
		tcancel()

		for {
			select {
			case <-opts.Ctx.Done():
				w.cancelSchedule()
				return
			default:
			}

			services, err := watcher.Next()
			if err != nil {
				continue
			}

			w.check(services)
		}
	}()
}

type watcher struct {
	opts  Options
	mu    sync.Mutex
	timer *time.Timer
}

func (w *watcher) check(services []*registry.ServiceInstance) {
	maxVersion := w.maxPeerVersion(services)
	if registry.CompareVersion(w.opts.Version, maxVersion) >= 0 {
		w.cancelSchedule()
		return
	}
	xlog.Sugar().Warnf("%s %s version %s is lower than cluster max %s, will retire in %v",
		w.opts.Kind, w.opts.ID, w.opts.Version, maxVersion, w.opts.RetireDelay)
	w.schedule()
}

func (w *watcher) maxPeerVersion(services []*registry.ServiceInstance) string {
	versions := make([]string, 0, len(services))
	for _, ins := range services {
		if ins == nil || ins.Kind != w.opts.Kind || ins.Alias != w.opts.Alias {
			continue
		}

		if w.opts.MatchGameID && ins.GameID != w.opts.GameID {
			continue
		}

		versions = append(versions, ins.Version)
	}

	if len(versions) == 0 {
		return w.opts.Version
	}

	return registry.MaxVersion(versions...)
}

func (w *watcher) schedule() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer != nil {
		return
	}

	w.timer = time.AfterFunc(w.opts.RetireDelay, func() {
		xlog.Sugar().Warnf("%s %s version %s retiring after %v", w.opts.Kind, w.opts.ID, w.opts.Version, w.opts.RetireDelay)

		w.opts.Shutdown()

		proc, err := os.FindProcess(os.Getpid())
		if err != nil {
			xlog.Sugar().Errorf("find process failed: %v", err)
			os.Exit(0)
			return
		}

		if err = proc.Signal(syscall.SIGTERM); err != nil {
			xlog.Sugar().Errorf("send retire signal failed: %v", err)
			os.Exit(0)
		}
	})
}

func (w *watcher) cancelSchedule() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timer == nil {
		return
	}

	w.timer.Stop()
	w.timer = nil
}
