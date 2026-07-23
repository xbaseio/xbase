package main

import (
	"context"
	"time"

	"github.com/xbaseio/xbase"
	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/cluster/node"
	"github.com/xbaseio/xbase/etc"
	"github.com/xbaseio/xbase/examples/cluster-demo/internal/bootstrap"
	"github.com/xbaseio/xbase/examples/cluster-demo/internal/observability"
	"github.com/xbaseio/xbase/registry"
)

type EchoRequest struct {
	Text string `json:"text"`
}

type RPCProfileRequest struct {
	UID int64 `json:"uid"`
}

type RPCProfileReply struct {
	UID           int64                  `json:"uid"`
	Name          string                 `json:"name"`
	From          string                 `json:"from"`
	ServiceStatus registry.ServiceStatus `json:"service_status"`
	Version       string                 `json:"version"`
}

type EchoReply struct {
	From          string                 `json:"from"`
	UID           int64                  `json:"uid"`
	GameID        int32                  `json:"game_id"`
	MessageID     int32                  `json:"message_id"`
	ServiceStatus registry.ServiceStatus `json:"service_status"`
	Version       string                 `json:"version"`
	Text          string                 `json:"text"`
	RPC           *RPCProfileReply       `json:"rpc,omitempty"`
}

type UserRPC struct {
	name          string
	version       string
	serviceStatus registry.ServiceStatus
}

func (s *UserRPC) GetProfile(ctx context.Context, req *RPCProfileRequest, resp *RPCProfileReply) error {
	resp.UID = req.UID
	resp.Name = "demo-user"
	resp.From = s.name
	resp.ServiceStatus = s.serviceStatus
	resp.Version = s.version

	return nil
}

func main() {
	locator := bootstrap.NewLocator()
	reg := bootstrap.NewRegistry()
	transporter := bootstrap.NewRPCXTransporter()
	debugServer := observability.NewDebugServer(
		etc.Get("etc.clusterDemo.debug.addr").String(),
		locator,
		reg,
	)

	n := node.NewNode(
		node.WithLocator(locator),
		node.WithRegistry(reg),
		node.WithTransporter(transporter),
	)

	serviceStatus := registry.ParseServiceStatus(etc.Get("etc.cluster.node.serviceStatus").String())
	version := etc.Get("etc.cluster.node.version", "1").String()
	name := etc.Get("etc.cluster.node.name", "node").String()

	n.Proxy().AddServiceProvider("user.rpc", "user.rpc", &UserRPC{
		name:          name,
		version:       version,
		serviceStatus: serviceStatus,
	})

	n.Proxy().AddEventHandler(cluster.Connect, func(ctx node.Context) {
		observability.Info(
			"node_connect",
			"uid", ctx.UID(),
			"cid", ctx.CID(),
			"gid", ctx.GID(),
			"nid", ctx.NID(),
			"nodeName", name,
			"version", version,
			"serviceStatus", serviceStatus,
		)
	})

	n.Proxy().AddEventHandler(cluster.Disconnect, func(ctx node.Context) {
		observability.Info(
			"node_disconnect",
			"uid", ctx.UID(),
			"cid", ctx.CID(),
			"gid", ctx.GID(),
			"nid", ctx.NID(),
			"nodeName", name,
			"version", version,
			"serviceStatus", serviceStatus,
		)
	})

	handlers := make(map[int32]node.MessageDispatcher)
	handlers[1001] = func(ctx node.Context) {
		started := time.Now()
		var req EchoRequest
		if err := ctx.Parse(&req); err != nil {
			observability.Warn("route_parse_failed", err,
				"uid", ctx.UID(),
				"gid", ctx.GID(),
				"nid", ctx.NID(),
				"gameID", ctx.GameID(),
				"messageID", ctx.MessageID(),
				"version", version,
				"serviceStatus", serviceStatus,
			)
			return
		}

		if err := ctx.Response(&EchoReply{
			From:          name,
			UID:           ctx.UID(),
			GameID:        ctx.GameID(),
			MessageID:     ctx.MessageID(),
			ServiceStatus: serviceStatus,
			Version:       version,
			Text:          req.Text,
		}); err != nil {
			observability.Warn("route_reply_failed", err,
				"uid", ctx.UID(),
				"gid", ctx.GID(),
				"nid", ctx.NID(),
				"gameID", ctx.GameID(),
				"messageID", ctx.MessageID(),
				"version", version,
				"serviceStatus", serviceStatus,
			)
			return
		}

		observability.Info("route_handled",
			"uid", ctx.UID(),
			"gid", ctx.GID(),
			"nid", ctx.NID(),
			"nodeName", name,
			"gameID", ctx.GameID(),
			"messageID", ctx.MessageID(),
			"routeType", "public",
			"latencyMs", time.Since(started).Milliseconds(),
			"version", version,
			"serviceStatus", serviceStatus,
		)
	}

	handlers[2001] = func(ctx node.Context) {
		started := time.Now()
		var req EchoRequest
		if err := ctx.Parse(&req); err != nil {
			observability.Warn("route_parse_failed", err,
				"uid", ctx.UID(),
				"gid", ctx.GID(),
				"nid", ctx.NID(),
				"gameID", ctx.GameID(),
				"messageID", ctx.MessageID(),
				"version", version,
				"serviceStatus", serviceStatus,
			)
			return
		}

		if err := ctx.Response(&EchoReply{
			From:          name,
			UID:           ctx.UID(),
			GameID:        ctx.GameID(),
			MessageID:     ctx.MessageID(),
			ServiceStatus: serviceStatus,
			Version:       version,
			Text:          "authorized:" + req.Text,
		}); err != nil {
			observability.Warn("route_reply_failed", err,
				"uid", ctx.UID(),
				"gid", ctx.GID(),
				"nid", ctx.NID(),
				"gameID", ctx.GameID(),
				"messageID", ctx.MessageID(),
				"version", version,
				"serviceStatus", serviceStatus,
			)
			return
		}

		observability.Info("route_handled",
			"uid", ctx.UID(),
			"gid", ctx.GID(),
			"nid", ctx.NID(),
			"nodeName", name,
			"gameID", ctx.GameID(),
			"messageID", ctx.MessageID(),
			"routeType", "business",
			"latencyMs", time.Since(started).Milliseconds(),
			"version", version,
			"serviceStatus", serviceStatus,
		)
	}

	handlers[3001] = func(ctx node.Context) {
		started := time.Now()
		var req EchoRequest
		if err := ctx.Parse(&req); err != nil {
			observability.Warn("route_parse_failed", err,
				"uid", ctx.UID(),
				"gid", ctx.GID(),
				"nid", ctx.NID(),
				"gameID", ctx.GameID(),
				"messageID", ctx.MessageID(),
				"version", version,
				"serviceStatus", serviceStatus,
			)
			return
		}

		cli, err := ctx.NewMeshClient("discovery://user.rpc")
		if err != nil {
			observability.Warn("rpc_client_create_failed", err,
				"uid", ctx.UID(),
				"service", "user.rpc",
				"target", "discovery://user.rpc",
				"version", version,
				"serviceStatus", serviceStatus,
			)
			return
		}

		rpcStarted := time.Now()
		rpcReply := &RPCProfileReply{}
		if err = cli.Call(
			ctx.Context(),
			"user.rpc",
			"GetProfile",
			&RPCProfileRequest{UID: ctx.UID()},
			rpcReply,
		); err != nil {
			observability.Warn("rpc_call_failed", err,
				"uid", ctx.UID(),
				"service", "user.rpc",
				"method", "GetProfile",
				"target", "discovery://user.rpc",
				"latencyMs", time.Since(rpcStarted).Milliseconds(),
				"version", version,
				"serviceStatus", serviceStatus,
			)
			return
		}

		observability.Info("rpc_call_succeeded",
			"uid", ctx.UID(),
			"service", "user.rpc",
			"method", "GetProfile",
			"target", "discovery://user.rpc",
			"targetNode", rpcReply.From,
			"targetVersion", rpcReply.Version,
			"targetServiceStatus", rpcReply.ServiceStatus,
			"latencyMs", time.Since(rpcStarted).Milliseconds(),
			"version", version,
			"serviceStatus", serviceStatus,
		)

		if err = ctx.Response(&EchoReply{
			From:          name,
			UID:           ctx.UID(),
			GameID:        ctx.GameID(),
			MessageID:     ctx.MessageID(),
			ServiceStatus: serviceStatus,
			Version:       version,
			Text:          "rpc:" + req.Text,
			RPC:           rpcReply,
		}); err != nil {
			observability.Warn("route_reply_failed", err,
				"uid", ctx.UID(),
				"gid", ctx.GID(),
				"nid", ctx.NID(),
				"gameID", ctx.GameID(),
				"messageID", ctx.MessageID(),
				"version", version,
				"serviceStatus", serviceStatus,
			)
			return
		}

		observability.Info("route_handled",
			"uid", ctx.UID(),
			"gid", ctx.GID(),
			"nid", ctx.NID(),
			"nodeName", name,
			"gameID", ctx.GameID(),
			"messageID", ctx.MessageID(),
			"routeType", "rpc",
			"latencyMs", time.Since(started).Milliseconds(),
			"version", version,
			"serviceStatus", serviceStatus,
		)
	}

	n.Proxy().BindMessageDispatcher(func(ctx node.Context) {
		handler, ok := handlers[ctx.MessageID()]
		if !ok {
			observability.Warn("message_not_supported", nil,
				"uid", ctx.UID(),
				"gameID", ctx.GameID(),
				"messageID", ctx.MessageID(),
			)
			return
		}
		handler(ctx)
	})

	n.Proxy().AddHookListener(cluster.Start, func(proxy *node.Proxy) {
		observability.Info(
			"node_started",
			"id", proxy.GetID(),
			"name", proxy.GetName(),
			"version", version,
			"serviceStatus", serviceStatus,
			"rpc", "user.rpc",
		)
	})

	container := xbase.NewContainer()
	container.Add(debugServer, n)
	container.Serve()
}
