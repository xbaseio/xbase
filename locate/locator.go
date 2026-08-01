package locate

import (
	"context"
)

type Locator interface {
	// Name 获取定位器组件名
	Name() string
	// Watch 监听用户定位变化
	Watch(ctx context.Context, kinds ...string) (Watcher, error)
	// BindGate 绑定网关
	BindGate(ctx context.Context, uid int64, gid string) error
	// BindNode 绑定节点
	BindNode(ctx context.Context, uid int64, name string, binding NodeBinding) error
	// UnbindGate 解绑网关
	UnbindGate(ctx context.Context, uid int64, gid string) error
	// UnbindNode 解绑节点
	UnbindNode(ctx context.Context, uid int64, name string, nid string) error
	// LocateGate 定位用户所在网关
	LocateGate(ctx context.Context, uid int64) (string, error)
	// LocateNode 定位用户所在节点
	LocateNode(ctx context.Context, uid int64, name string) (string, error)
	// LocateNodeBinding locates the node and its per-user binding metadata.
	LocateNodeBinding(ctx context.Context, uid int64, name string) (NodeBinding, error)
	// LocateNodes 定位用户所在节点列表
	LocateNodes(ctx context.Context, uid int64) (map[string]string, error)
}

// NodeBinding describes the node selected for a user and the metadata that
// should accompany messages routed through this binding.
type NodeBinding struct {
	NID      string            `json:"nid"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Watcher interface {
	// Next 返回用户位置列表
	Next() ([]*Event, error)
	// Stop 停止监听
	Stop() error
}

type Event struct {
	// 用户ID
	UID int64 `json:"uid"`
	// 事件类型
	Type EventType `json:"type"`
	// 实例ID
	InsID string `json:"insID"`
	// 实例类型
	InsKind string `json:"insKind"`
	// 实例名称
	InsName string `json:"insName"`
	// Metadata contains the per-user node binding metadata.
	Metadata map[string]string `json:"metadata,omitempty"`
}

type EventType int

const (
	BindGate   EventType = iota + 1 // 绑定网关
	BindNode                        // 绑定节点
	UnbindGate                      // 解绑网关
	UnbindNode                      // 解绑节点
)
