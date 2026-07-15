package bootstrap

import (
	"github.com/xbaseio/xbase/eventbus"
	natseventbus "github.com/xbaseio/xbase/eventbus/nats"
	"github.com/xbaseio/xbase/locate"
	redislocate "github.com/xbaseio/xbase/locate/redis"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/network/tcp"
	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/registry/consul"
	"github.com/xbaseio/xbase/transport"
	grpctransport "github.com/xbaseio/xbase/transport/grpc"
	rpcxtransport "github.com/xbaseio/xbase/transport/rpcx"
)

func NewLocator() locate.Locator {
	return redislocate.NewLocator()
}

func NewRegistry() registry.Registry {
	return consul.NewRegistry()
}

func NewGateServer() network.Server {
	return tcp.NewServer()
}

func NewTransporter() transport.Transporter {
	return grpctransport.NewTransporter()
}

func NewRPCXTransporter() transport.Transporter {
	return rpcxtransport.NewTransporter()
}

func NewEventbus() eventbus.Eventbus {
	return natseventbus.NewEventbus()
}
