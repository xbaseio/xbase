package file

import (
	"github.com/xbaseio/xbase/config"
	"github.com/xbaseio/xbase/config/file/core"
	"github.com/xbaseio/xbase/log"
)

const Name = core.Name

type Source struct {
	opts *options
}

func NewSource(opts ...Option) config.Source {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	if o.path == "" {
		log.Fatal("no config file path specified")
	}

	return core.NewSource(o.path, o.mode)
}
