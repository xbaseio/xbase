package gate

import (
	"context"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/internal/link"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/mode"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/packet"
	"github.com/xbaseio/xbase/xerrors"
)

type proxy struct {
	gate         *Gate            // 网关服
	nodeLinker   *link.NodeLinker // 节点链接器
	routeWatcher *routeWatcher    // 路由策略
}

func newProxy(gate *Gate) *proxy {
	return &proxy{
		gate: gate,
		nodeLinker: link.NewNodeLinker(gate.ctx, &link.Options{
			InsID:            gate.opts.id,
			InsKind:          cluster.Gate,
			Locator:          gate.opts.locator,
			Registry:         gate.opts.registry,
			Dispatch:         gate.opts.dispatch,
			NodeKind:         gate.opts.nodeKind,
			GameID:           gate.opts.gameID,
			AllowTestService: gate.opts.allowTestService,
		}),
		routeWatcher: newRouteWatcher(gate.ctx, gate.opts.registry),
	}
}

// 绑定用户与网关间的关系
func (p *proxy) bindGate(ctx context.Context, cid, uid int64) error {
	err := p.gate.opts.locator.BindGate(ctx, uid, p.gate.opts.id)
	if err != nil {
		return err
	}

	p.trigger(ctx, cluster.Reconnect, cid, uid)

	return nil
}

// 解绑用户与网关间的关系
func (p *proxy) unbindGate(ctx context.Context, cid, uid int64) error {
	err := p.gate.opts.locator.UnbindGate(ctx, uid, p.gate.opts.id)
	if err != nil {
		log.Errorf("user unbind failed, gid: %s, cid: %d, uid: %d, err: %v", p.gate.opts.id, cid, uid, err)
	}

	return err
}

// 触发事件
func (p *proxy) trigger(ctx context.Context, event cluster.Event, cid, uid int64) {
	if mode.IsDebugMode() {
		log.Debugf("trigger event, event: %v cid: %d uid: %d", event.String(), cid, uid)
	}

	if err := p.nodeLinker.Trigger(ctx, &link.TriggerArgs{
		Event: event,
		CID:   cid,
		UID:   uid,
	}); err != nil {
		switch {
		case xerrors.Is(err, xerrors.ErrNotFoundEvent), xerrors.Is(err, xerrors.ErrNotFoundUserLocation):
			log.Warnf("trigger event failed, cid: %d, uid: %d, event: %v, err: %v", cid, uid, event.String(), err)
		default:
			log.Errorf("trigger event failed, cid: %d, uid: %d, event: %v, err: %v", cid, uid, event.String(), err)
		}
	}
}

// 投递消息
func (p *proxy) deliver(ctx context.Context, conn network.Conn, cid, uid int64, data []byte) {
	message, _, err := packet.UnpackMessage(data)
	if err != nil {
		log.Errorf("unpack message failed: %v", err)
		return
	}
	if message == nil {
		log.Warnf("unpack message failed: half packet")

		return
	}

	if p.isLoginMessage(message) {
		p.handleLogin(ctx, conn, message)
		return
	}

	if uid == 0 && p.routeWatcher.requiresAuth(message.GameID, message.MessageID) {
		log.Warnf("unauthorized route rejected, cid: %d message: %d game: %d", cid, message.MessageID, message.GameID)
		return
	}

	if err = p.nodeLinker.Deliver(ctx, &link.DeliverArgs{
		CID:       cid,
		UID:       uid,
		GameID:    message.GameID,
		MessageID: message.MessageID,
		Buffer:    data,
	}); err != nil {
		switch {
		case xerrors.Is(err, xerrors.ErrNotFoundRoute), xerrors.Is(err, xerrors.ErrNotFoundEndpoint):
			log.Warnf("deliver message failed, cid: %d uid: %d seq: %d game: %d message: %d err: %v", cid, uid, message.Seq, message.GameID, message.MessageID, err)
		default:
			log.Errorf("deliver message failed, cid: %d uid: %d seq: %d game: %d message: %d err: %v", cid, uid, message.Seq, message.GameID, message.MessageID, err)
		}
	} else {
		if mode.IsDebugMode() {
			log.Debugf("deliver message success, cid: %d uid: %d seq: %d game: %d message: %d", cid, uid, message.Seq, message.GameID, message.MessageID)
		}
	}
}

// 开始监听
func (p *proxy) watch() {
	p.routeWatcher.start()

	p.nodeLinker.WatchUserLocate()

	p.nodeLinker.WatchClusterInstance()
}
