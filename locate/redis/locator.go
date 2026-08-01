package redis

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/core/tls"
	"github.com/xbaseio/xbase/encoding/json"
	"github.com/xbaseio/xbase/locate"
	"github.com/xbaseio/xbase/xerrors"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	userGateKey     = "%s:locate:user:%d:gate"     // string
	userNodeKey     = "%s:locate:user:%d:node"     // hash
	clusterEventKey = "%s:locate:cluster:%s:event" // channel
)

const name = "redis"

var _ locate.Locator = &Locator{}

type Locator struct {
	err              error
	opts             *options
	builtin          bool
	ctx              context.Context
	cancel           context.CancelFunc
	sfg              singleflight.Group
	watchers         sync.Map
	unbindGateScript *redis.Script
	unbindNodeScript *redis.Script
}

func NewLocator(opts ...Option) *Locator {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	l := &Locator{}

	defer func() {
		if l.err == nil {
			l.opts = o
			l.ctx, l.cancel = context.WithCancel(o.ctx)
			l.unbindGateScript = redis.NewScript(unbindGateScript)
			l.unbindNodeScript = redis.NewScript(unbindNodeScript)
		}
	}()

	if o.client == nil {
		options := &redis.UniversalOptions{
			Addrs:      o.addrs,
			DB:         o.db,
			Username:   o.username,
			Password:   o.password,
			MaxRetries: o.maxRetries,
		}

		if o.certFile != "" && o.keyFile != "" && o.caFile != "" {
			if options.TLSConfig, l.err = tls.MakeRedisTLSConfig(o.certFile, o.keyFile, o.caFile); l.err != nil {
				return l
			}
		}

		o.client, l.builtin = redis.NewUniversalClient(options), true
	}

	return l
}

// Name 获取定位器组件名
func (l *Locator) Name() string {
	return name
}

// LocateGate 定位用户所在网关
func (l *Locator) LocateGate(ctx context.Context, uid int64) (string, error) {
	if l.err != nil {
		return "", l.err
	}

	key := fmt.Sprintf(userGateKey, l.opts.prefix, uid)

	val, err, _ := l.sfg.Do(key, func() (any, error) {
		val, err := l.opts.client.Get(ctx, key).Result()
		if err != nil && !xerrors.Is(err, redis.Nil) {
			return "", err
		}

		return val, nil
	})
	if err != nil {
		return "", err
	}

	return val.(string), nil
}

// LocateNode 定位用户所在节点
func (l *Locator) LocateNode(ctx context.Context, uid int64, name string) (string, error) {
	binding, err := l.LocateNodeBinding(ctx, uid, name)
	return binding.NID, err
}

// LocateNodeBinding locates the node and its per-user binding metadata.
func (l *Locator) LocateNodeBinding(ctx context.Context, uid int64, name string) (locate.NodeBinding, error) {
	if l.err != nil {
		return locate.NodeBinding{}, l.err
	}

	key := fmt.Sprintf(userNodeKey, l.opts.prefix, uid)

	val, err, _ := l.sfg.Do(key+name, func() (any, error) {
		val, err := l.opts.client.HGet(ctx, key, name).Result()
		if err != nil && !xerrors.Is(err, redis.Nil) {
			return locate.NodeBinding{}, err
		}

		return unmarshalNodeBinding(val)
	})
	if err != nil {
		return locate.NodeBinding{}, err
	}

	return cloneNodeBinding(val.(locate.NodeBinding)), nil
}

// LocateNodes 定位用户所在节点列表
func (l *Locator) LocateNodes(ctx context.Context, uid int64) (map[string]string, error) {
	if l.err != nil {
		return nil, l.err
	}

	key := fmt.Sprintf(userNodeKey, l.opts.prefix, uid)

	val, err, _ := l.sfg.Do(key, func() (any, error) {
		val, err := l.opts.client.HGetAll(ctx, key).Result()
		if err != nil && !xerrors.Is(err, redis.Nil) {
			return nil, err
		}

		bindings := make(map[string]string, len(val))
		for name, encoded := range val {
			binding, err := unmarshalNodeBinding(encoded)
			if err != nil {
				return nil, err
			}
			bindings[name] = binding.NID
		}

		return bindings, nil
	})
	if err != nil {
		return nil, err
	}

	return val.(map[string]string), nil
}

// BindGate 绑定网关
func (l *Locator) BindGate(ctx context.Context, uid int64, gid string) error {
	if l.err != nil {
		return l.err
	}

	key := fmt.Sprintf(userGateKey, l.opts.prefix, uid)

	if err := l.opts.client.Set(ctx, key, gid, redis.KeepTTL).Err(); err != nil {
		return err
	}

	if err := l.broadcast(ctx, locate.BindGate, uid, gid, nil); err != nil {
		xlog.Logger().Error("location event broadcast failed", zap.Error(err))
	}

	return nil
}

// BindNode 绑定节点
func (l *Locator) BindNode(ctx context.Context, uid int64, name string, binding locate.NodeBinding) error {
	if l.err != nil {
		return l.err
	}
	if binding.NID == "" {
		return xerrors.ErrInvalidNID
	}

	key := fmt.Sprintf(userNodeKey, l.opts.prefix, uid)
	binding = cloneNodeBinding(binding)
	encoded, err := marshalNodeBinding(binding)
	if err != nil {
		return err
	}

	if err := l.opts.client.HSet(ctx, key, name, encoded).Err(); err != nil {
		return err
	}

	if err := l.broadcast(ctx, locate.BindNode, uid, binding.NID, binding.Metadata, name); err != nil {
		xlog.Logger().Error("location event broadcast failed", zap.Error(err))
	}

	return nil
}

// UnbindGate 解绑网关
func (l *Locator) UnbindGate(ctx context.Context, uid int64, gid string) error {
	if l.err != nil {
		return l.err
	}

	key := fmt.Sprintf(userGateKey, l.opts.prefix, uid)

	rst, err := l.unbindGateScript.Run(ctx, l.opts.client, []string{key}, gid).StringSlice()
	if err != nil {
		return err
	}

	if rst[0] == "OK" {
		if err = l.broadcast(ctx, locate.UnbindGate, uid, gid, nil); err != nil {
			xlog.Logger().Error("location event broadcast failed", zap.Error(err))
		}
	}

	return nil
}

// UnbindNode 解绑节点
func (l *Locator) UnbindNode(ctx context.Context, uid int64, name, nid string) error {
	if l.err != nil {
		return l.err
	}

	key := fmt.Sprintf(userNodeKey, l.opts.prefix, uid)

	rst, err := l.unbindNodeScript.Run(ctx, l.opts.client, []string{key}, name, nid).StringSlice()
	if err != nil {
		return err
	}

	if rst[0] == "OK" {
		if err = l.broadcast(ctx, locate.UnbindNode, uid, nid, nil, name); err != nil {
			xlog.Logger().Error("location event broadcast failed", zap.Error(err))
		}
	}

	return nil
}

// 广播事件
func (l *Locator) broadcast(ctx context.Context, typ locate.EventType, uid int64, insID string, metadata map[string]string, insName ...string) error {
	evt := &locate.Event{UID: uid, Type: typ, InsID: insID, Metadata: maps.Clone(metadata)}

	switch typ {
	case locate.BindGate, locate.UnbindGate:
		evt.InsKind = cluster.Gate.String()
	case locate.BindNode, locate.UnbindNode:
		evt.InsKind = cluster.Node.String()
	}

	if len(insName) > 0 {
		evt.InsName = insName[0]
	}

	msg, err := marshal(evt)
	if err != nil {
		return err
	}

	return l.opts.client.Publish(ctx, fmt.Sprintf(clusterEventKey, l.opts.prefix, evt.InsKind), msg).Err()
}

func (l *Locator) toUniqueKey(kinds ...string) string {
	slices.Sort(kinds)

	return strings.Join(kinds, "&")
}

// Watch 监听用户定位变化
func (l *Locator) Watch(ctx context.Context, kinds ...string) (locate.Watcher, error) {
	if l.err != nil {
		return nil, l.err
	}

	key := l.toUniqueKey(kinds...)

	v, ok := l.watchers.Load(key)
	if ok {
		return v.(*watcherMgr).fork(), nil
	}

	w, err := newWatcherMgr(ctx, l, key, kinds...)
	if err != nil {
		return nil, err
	}

	l.watchers.Store(key, w)

	return w.fork(), nil
}

func marshal(event *locate.Event) (string, error) {
	buf, err := json.Marshal(event)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

func unmarshal(data []byte) (*locate.Event, error) {
	evt := &locate.Event{}

	if err := json.Unmarshal(data, evt); err != nil {
		return nil, err
	}

	return evt, nil
}

func marshalNodeBinding(binding locate.NodeBinding) (string, error) {
	buf, err := json.Marshal(binding)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func unmarshalNodeBinding(data string) (locate.NodeBinding, error) {
	if data == "" {
		return locate.NodeBinding{}, nil
	}

	// Bindings written before metadata support contain the raw node ID.
	if !strings.HasPrefix(data, "{") {
		return locate.NodeBinding{NID: data}, nil
	}

	binding := locate.NodeBinding{}
	if err := json.Unmarshal([]byte(data), &binding); err != nil {
		return locate.NodeBinding{}, err
	}
	return binding, nil
}

func cloneNodeBinding(binding locate.NodeBinding) locate.NodeBinding {
	binding.Metadata = maps.Clone(binding.Metadata)
	return binding
}
