package link

import (
	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/crypto"
	"github.com/xbaseio/xbase/encoding"
	"github.com/xbaseio/xbase/locate"
	"github.com/xbaseio/xbase/registry"
)

type Options struct {
	InsID                string
	InsKind              cluster.Kind
	Codec                encoding.Codec
	Locator              locate.Locator
	Registry             registry.Registry
	Encryptor            crypto.Encryptor
	Dispatch             cluster.Dispatch
	NodeKind             cluster.NodeKind
	GameID               int32
	WaitHandler          func()
	DoneHandler          func()
	ResolveServiceStatus func(uid int64) registry.ServiceStatus
}
