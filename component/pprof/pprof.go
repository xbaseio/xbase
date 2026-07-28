package pprof

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"

	"github.com/xbaseio/xbase/component"
	"github.com/xbaseio/xbase/core/info"
	xnet "github.com/xbaseio/xbase/core/net"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

var _ component.Component = &PProf{}

type PProf struct {
	component.Base
	opts *options
}

func NewPProf(opts ...Option) *PProf {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	return &PProf{opts: o}
}

func (*PProf) Name() string {
	return "pprof"
}

func (p *PProf) Start() {
	listenAddr, exposeAddr, err := xnet.ParseAddr(p.opts.addr)
	if err != nil {
		xlog.Logger().Fatal("pprof addr parse failed", zap.Error(err))
	}

	go func() {
		if err := http.ListenAndServe(listenAddr, nil); err != nil {
			xlog.Logger().Fatal("pprof server start failed", zap.Error(err))
		}
	}()

	info.PrintBoxInfo("PProf",
		fmt.Sprintf("Url: http://%s/debug/pprof/", exposeAddr),
	)
}
