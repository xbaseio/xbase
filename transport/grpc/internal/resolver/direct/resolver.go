package direct

import (
	"github.com/xbaseio/xbase/xerrors"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"
)

type Resolver struct {
	builder *Builder
	target  resolver.Target
	cc      resolver.ClientConn
}

func (r *Resolver) ResolveNow(_ resolver.ResolveNowOptions) {
	if r.builder != nil {
		r.builder.updateResolver(r)
	}
}

func (r *Resolver) Close() {
	if r.builder != nil {
		r.builder.removeResolver(r)
	}
}

func (r *Resolver) updateState(state resolver.State) {
	if err := r.cc.UpdateState(state); err != nil {
		r.cc.ReportError(err)

		if !(len(state.Addresses) == 0 && xerrors.Is(err, balancer.ErrBadResolverState)) {
			xlog.Logger().Warn("update client conn state failed", zap.Error(err))
		}
	}
}
