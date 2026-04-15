package link

import (
	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/crypto"
	"github.com/xbaseio/xbase/encoding"
	"github.com/xbaseio/xbase/locate"
	"github.com/xbaseio/xbase/registry"
)

type Options struct {
	InsID       string            // 实例ID
	InsKind     cluster.Kind      // 实例类型
	Codec       encoding.Codec    // 编解码器
	Locator     locate.Locator    // 定位器
	Registry    registry.Registry // 注册器
	Encryptor   crypto.Encryptor  // 加密器
	Dispatch    cluster.Dispatch  // 无状态路由消息分发策略
	WaitHandler func()            // 等待处理
	DoneHandler func()            // 完成处理
}
