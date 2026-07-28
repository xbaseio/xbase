package gate

import (
	"context"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/internal/link"
	"github.com/xbaseio/xbase/mode"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/packet"
	"github.com/xbaseio/xbase/utils/xcall"
	"github.com/xbaseio/xbase/xerrors"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

type proxy struct {
	gate       *Gate            // 网关服
	nodeLinker *link.NodeLinker // 节点链接器
}

func newProxy(gate *Gate) *proxy {
	return &proxy{
		gate: gate,
		nodeLinker: link.NewNodeLinker(gate.ctx, &link.Options{
			InsID:                gate.opts.id,
			InsKind:              cluster.Gate,
			Locator:              gate.opts.locator,
			Registry:             gate.opts.registry,
			Dispatch:             gate.opts.dispatch,
			NodeKind:             gate.opts.nodeKind,
			GameID:               gate.opts.gameID,
			ResolveServiceStatus: gate.resolveServiceStatus,
		}),
	}
}

// 绑定用户与网关间的关系
func (p *proxy) bindGate(ctx context.Context, cid, uid int64) error {
	return p.gate.opts.locator.BindGate(ctx, uid, p.gate.opts.id)
}

// bindLobby binds the user to the lobby route when the cluster has one.
func (p *proxy) bindLobby(ctx context.Context, uid int64) error {
	return p.nodeLinker.BindGameNode(ctx, uid, cluster.LobbyGameID)
}

// 解绑用户与网关间的关系
func (p *proxy) unbindGate(ctx context.Context, cid, uid int64) error {
	err := p.gate.opts.locator.UnbindGate(ctx, uid, p.gate.opts.id)
	if err != nil {
		xlog.Logger().Error("user unbind failed, gid: , cid: , uid: , err", zap.Any("id", p.gate.opts.id), zap.Any("cid", cid), zap.Any("uid", uid), zap.Error(err))
	}

	return err
}

// 触发事件
func (p *proxy) trigger(ctx context.Context, event cluster.Event, cid, uid int64) {
	if mode.IsDebugMode() {
		xlog.Logger().Debug("trigger event, event: cid: uid", zap.String("event", event.String()), zap.Any("cid", cid), zap.Any("uid", uid))
	}

	if err := p.nodeLinker.Trigger(ctx, &link.TriggerArgs{
		Event: event,
		CID:   cid,
		UID:   uid,
	}); err != nil {
		switch {
		case xerrors.Is(err, xerrors.ErrNotFoundEvent), xerrors.Is(err, xerrors.ErrNotFoundUserLocation):
			xlog.Logger().Warn("trigger event failed, cid: , uid: , event: , err", zap.Any("cid", cid), zap.Any("uid", uid), zap.String("event", event.String()), zap.Error(err))
		default:
			xlog.Logger().Error("trigger event failed, cid: , uid: , event: , err", zap.Any("cid", cid), zap.Any("uid", uid), zap.String("event", event.String()), zap.Error(err))
		}
	}
}

// 投递消息
func (p *proxy) deliver(ctx context.Context, conn network.Conn, cid, uid int64, data []byte) {
	message, _, err := packet.UnpackMessage(data)
	if err != nil {
		xlog.Logger().Error("unpack message failed", zap.Error(err))
		return
	}
	if message == nil {
		xlog.Logger().Warn("unpack message failed: half packet")

		return
	}

	if message.GameID == cluster.GateGameID {
		p.dispatchGateMessage(ctx, conn, message)
		return
	}

	if uid == 0 {
		xlog.Logger().Warn("unbound connection message rejected, cid: message: game", zap.Any("cid", cid), zap.Any("messageID", message.MessageID), zap.Any("gameID", message.GameID))
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
			xlog.Logger().Warn("deliver message failed, cid: uid: seq: game: message: err", zap.Any("cid", cid), zap.Any("uid", uid), zap.Any("seq", message.Seq), zap.Any("gameID", message.GameID), zap.Any("messageID", message.MessageID), zap.Error(err))
		default:
			xlog.Logger().Error("deliver message failed, cid: uid: seq: game: message: err", zap.Any("cid", cid), zap.Any("uid", uid), zap.Any("seq", message.Seq), zap.Any("gameID", message.GameID), zap.Any("messageID", message.MessageID), zap.Error(err))
		}
	} else {
		if mode.IsDebugMode() {
			xlog.Logger().Debug("deliver message success, cid: uid: seq: game: message", zap.Any("cid", cid), zap.Any("uid", uid), zap.Any("seq", message.Seq), zap.Any("gameID", message.GameID), zap.Any("messageID", message.MessageID))
		}
	}
}

// dispatchGateMessage sends Gate control messages directly to the upper layer.
func (p *proxy) dispatchGateMessage(ctx context.Context, conn network.Conn, message *packet.Message) {
	dispatcher := p.gate.opts.messageDispatcher
	if dispatcher == nil {
		xlog.Logger().Warn("gate message dispatcher is not bound, message", zap.Any("messageID", message.MessageID))
		return
	}

	xcall.Call(func() {
		dispatcher(&messageContext{ctx: ctx, gate: p.gate, conn: conn, message: message})
	})
}

// 开始监听
func (p *proxy) watch() {
	p.nodeLinker.WatchUserLocate()

	p.nodeLinker.WatchClusterInstance()
}
