package gate

import (
	"context"
	"strings"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/encoding/json"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/packet"
	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/utils/xconv"
	"github.com/xbaseio/xbase/utils/xjwt"
	"github.com/xbaseio/xbase/xerrors"
)

const (
	loginCodeOK           = 0
	loginCodeInvalidToken = 1
	loginCodeInvalidUID   = 2
	loginCodeBindFailed   = 3
	loginCodeLobbyFailed  = 4
)

type loginRequest struct {
	Token string `json:"token"`
}

type loginReply struct {
	Code int    `json:"code"`
	UID  int64  `json:"uid,omitempty"`
	Msg  string `json:"msg,omitempty"`
}

func (p *proxy) isLoginMessage(message *packet.Message) bool {
	if p.gate.opts.jwt == nil || p.gate.opts.loginMessageID <= 0 {
		return false
	}

	return message.GameID == p.gate.opts.lobbyGameID &&
		message.MessageID == p.gate.opts.loginMessageID
}

func (p *proxy) handleLogin(ctx context.Context, conn network.Conn, message *packet.Message) {
	cid := conn.ID()

	token := parseLoginToken(message.Buffer)
	if token == "" {
		p.pushLoginReply(conn, message.Seq, loginCodeInvalidToken, 0, "missing token")
		return
	}

	identity, err := p.gate.opts.jwt.ExtractIdentity(token)
	if err != nil {
		log.Warnf("login token invalid, cid: %d err: %v", cid, err)
		p.pushLoginReply(conn, message.Seq, loginCodeInvalidToken, 0, err.Error())
		return
	}

	uid := xconv.Int64(identity)
	if uid <= 0 {
		p.pushLoginReply(conn, message.Seq, loginCodeInvalidUID, 0, "invalid uid")
		return
	}

	if err = p.gate.session.Bind(cid, uid); err != nil {
		log.Errorf("login bind session failed, cid: %d uid: %d err: %v", cid, uid, err)
		p.pushLoginReply(conn, message.Seq, loginCodeBindFailed, 0, err.Error())
		return
	}

	if err = p.bindGate(ctx, cid, uid); err != nil {
		_, _ = p.gate.session.Unbind(cid, uid)
		log.Errorf("login bind gate failed, cid: %d uid: %d err: %v", cid, uid, err)
		p.pushLoginReply(conn, message.Seq, loginCodeBindFailed, 0, err.Error())
		return
	}

	if err = p.bindLobbyIfAbsent(ctx, uid); err != nil {
		log.Errorf("login bind lobby failed, cid: %d uid: %d err: %v", cid, uid, err)
		p.pushLoginReply(conn, message.Seq, loginCodeLobbyFailed, uid, err.Error())
		return
	}

	p.pushLoginReply(conn, message.Seq, loginCodeOK, uid, "ok")
}

func parseLoginToken(body []byte) string {
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		return ""
	}

	if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return strings.TrimSpace(raw[7:])
	}

	var req loginRequest
	if err := json.Unmarshal(body, &req); err == nil && strings.TrimSpace(req.Token) != "" {
		token := strings.TrimSpace(req.Token)
		if strings.HasPrefix(strings.ToLower(token), "bearer ") {
			return strings.TrimSpace(token[7:])
		}
		return token
	}

	return raw
}

func (p *proxy) bindLobbyIfAbsent(ctx context.Context, uid int64) error {
	ins, err := p.pickLobbyNode(ctx, uid)
	if err != nil {
		return err
	}

	name := ins.Alias
	if name == "" {
		name = ins.Name
	}

	if _, err = p.nodeLinker.LocateNode(ctx, uid, name); err == nil {
		return nil
	} else if !xerrors.Is(err, xerrors.ErrNotFoundUserLocation) {
		return err
	}

	return p.nodeLinker.BindNode(ctx, uid, name, ins.ID)
}

func (p *proxy) pickLobbyNode(ctx context.Context, uid int64) (*registry.ServiceInstance, error) {
	services, err := p.gate.opts.registry.Services(ctx, cluster.Node.String())
	if err != nil {
		return nil, err
	}

	maxVersion := registry.MaxVersionForGame(services)
	candidates := make([]*registry.ServiceInstance, 0, len(services))

	for _, ins := range services {
		if ins == nil || ins.Kind != cluster.Node.String() {
			continue
		}

		if ins.GameID != p.gate.opts.lobbyGameID {
			continue
		}

		switch ins.State {
		case cluster.Work.String(), cluster.Busy.String():
		default:
			continue
		}

		if !registry.IsLatestVersion(ins, maxVersion[ins.GameID]) {
			continue
		}

		candidates = append(candidates, ins)
	}

	if len(candidates) == 0 {
		return nil, xerrors.ErrNotFoundEndpoint
	}

	idx := uid % int64(len(candidates))
	if idx < 0 {
		idx = -idx
	}

	return candidates[idx], nil
}

func (p *proxy) pushLoginReply(conn network.Conn, seq int32, code int, uid int64, msg string) {
	body, err := json.Marshal(&loginReply{
		Code: code,
		UID:  uid,
		Msg:  msg,
	})
	if err != nil {
		log.Errorf("login reply marshal failed: %v", err)
		return
	}

	wire, err := packet.PackMessage(&packet.Message{
		Seq:       seq,
		GameID:    p.gate.opts.lobbyGameID,
		MessageID: p.gate.opts.loginMessageID,
		Buffer:    body,
	})
	if err != nil {
		log.Errorf("login reply pack failed: %v", err)
		return
	}

	if err = conn.Push(wire); err != nil {
		log.Errorf("login reply push failed, cid: %d err: %v", conn.ID(), err)
	}
}

func initGateJWT(opts *options) error {
	if opts.jwt != nil {
		return nil
	}

	secret := opts.jwtSecretKey
	if secret == "" {
		return nil
	}

	identityKey := opts.jwtIdentityKey
	if identityKey == "" {
		identityKey = defaultJWTIdentityKey
	}

	signAlgorithm := opts.jwtSignAlgorithm
	if signAlgorithm == "" {
		signAlgorithm = xjwt.HS256
	}

	jwt, err := xjwt.NewJWT(
		xjwt.WithSecretKey(secret),
		xjwt.WithIdentityKey(identityKey),
		xjwt.WithSignAlgorithm(signAlgorithm),
	)
	if err != nil {
		return err
	}

	opts.jwt = jwt
	return nil
}
