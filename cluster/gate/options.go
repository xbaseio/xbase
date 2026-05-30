package gate

import (
	"context"
	"maps"
	"runtime"
	"time"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/etc"
	"github.com/xbaseio/xbase/locate"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/network"
	"github.com/xbaseio/xbase/registry"
	"github.com/xbaseio/xbase/utils/xjwt"
	"github.com/xbaseio/xbase/utils/xuuid"
)

const (
	defaultName           = "gate"          // 默认名称
	defaultAddr           = ":0"            // 连接器监听地址
	defaultTimeout        = 3 * time.Second // 默认超时时间
	defaultDispatch       = cluster.Random  // 默认的无状态路由分发策略
	defaultVersion        = "1"
	defaultRetireDelay    = 10 * time.Minute
	defaultReceiveQueue   = 8192
	defaultLoginMessageID = 1000
	defaultLobbyGameID    = 0
	defaultJWTIdentityKey = "uid"
)

const (
	defaultIDKey       = "etc.cluster.gate.id"
	defaultNameKey     = "etc.cluster.gate.name"
	defaultAddrKey     = "etc.cluster.gate.addr"
	defaultExposeKey   = "etc.cluster.gate.expose"
	defaultTimeoutKey  = "etc.cluster.gate.timeout"
	defaultDispatchKey    = "etc.cluster.gate.dispatch"
	defaultMetadataKey    = "etc.cluster.gate.metadata"
	defaultVersionKey     = "etc.cluster.gate.version"
	defaultRetireDelayKey = "etc.cluster.gate.retireDelay"
	defaultReceiveQueueKey   = "etc.cluster.gate.receiveQueue"
	defaultDeliverWorkersKey = "etc.cluster.gate.deliverWorkers"
	defaultLoginMessageIDKey = "etc.cluster.gate.login.messageID"
	defaultLobbyGameIDKey    = "etc.cluster.gate.login.lobbyGameID"
	defaultJWTSecretKey      = "etc.cluster.gate.jwt.secretKey"
	defaultJWTIdentityKeyKey = "etc.cluster.gate.jwt.identityKey"
	defaultJWTSignAlgorithmKey = "etc.cluster.gate.jwt.signAlgorithm"
)

type Option func(o *options)

type options struct {
	ctx      context.Context   // 上下文
	id       string            // 实例ID
	name     string            // 实例名称
	addr     string            // 监听地址
	expose   bool              // 是否将内部通信地址暴露到公网
	timeout  time.Duration     // RPC调用超时时间
	server   network.Server    // 网关服务器
	locator  locate.Locator    // 用户定位器
	registry registry.Registry // 服务注册器
	dispatch cluster.Dispatch  // 无状态路由消息分发策略
	nodeKind cluster.NodeKind  // 节点类型
	gameID   int32             // 游戏ID
	version  string            // 服务版本号
	retireDelay time.Duration  // 低版本退出等待时间
	receiveQueue   int           // 收包队列容量
	deliverWorkers int           // deliver worker 数量
	loginMessageID int32         // Gate 登录消息 ID，<=0 表示关闭
	lobbyGameID    int32         // 大厅 GameID，默认 0
	jwt            *xjwt.JWT     // JWT 解析器
	jwtSecretKey   string        // JWT 密钥（etc 注入）
	jwtIdentityKey string        // JWT 用户 ID 字段名
	jwtSignAlgorithm xjwt.SignAlgorithm // JWT 签名算法
	metadata map[string]string // 元数据
}

func defaultOptions() *options {
	opts := &options{
		ctx:      context.Background(),
		name:     defaultName,
		addr:     defaultAddr,
		timeout:  defaultTimeout,
		dispatch: defaultDispatch,
		metadata: make(map[string]string),
		expose:   etc.Get(defaultExposeKey).Bool(),
		nodeKind:    cluster.Node_Normal,
		gameID:      -1,
		version:     defaultVersion,
		retireDelay: defaultRetireDelay,
		receiveQueue:   defaultReceiveQueue,
		deliverWorkers: max(runtime.NumCPU(), 1),
		loginMessageID: defaultLoginMessageID,
		lobbyGameID:    defaultLobbyGameID,
		jwtIdentityKey: defaultJWTIdentityKey,
		jwtSignAlgorithm: xjwt.HS256,
	}

	if id := etc.Get(defaultIDKey).String(); id != "" {
		opts.id = id
	} else {
		opts.id = xuuid.UUID()
	}

	if name := etc.Get(defaultNameKey).String(); name != "" {
		opts.name = name
	}

	if addr := etc.Get(defaultAddrKey).String(); addr != "" {
		opts.addr = addr
	}

	if timeout := etc.Get(defaultTimeoutKey).Duration(); timeout > 0 {
		opts.timeout = timeout
	}

	if strategy := etc.Get(defaultDispatchKey).String(); strategy != "" {
		opts.dispatch = cluster.Dispatch(strategy)
	}

	if err := etc.Get(defaultMetadataKey).Scan(&opts.metadata); err != nil {
		log.Warnf("scan gate metadata failed: %v", err)
	}

	if version := etc.Get(defaultVersionKey).String(); version != "" {
		opts.version = version
	}

	if delay := etc.Get(defaultRetireDelayKey).Duration(); delay > 0 {
		opts.retireDelay = delay
	}

	if n := etc.Get(defaultReceiveQueueKey).Int(); n > 0 {
		opts.receiveQueue = n
	}

	if n := etc.Get(defaultDeliverWorkersKey).Int(); n > 0 {
		opts.deliverWorkers = n
	}

	if etc.Has(defaultLoginMessageIDKey) {
		opts.loginMessageID = int32(etc.Get(defaultLoginMessageIDKey).Int())
	}

	if etc.Has(defaultLobbyGameIDKey) {
		opts.lobbyGameID = int32(etc.Get(defaultLobbyGameIDKey).Int())
	}

	opts.jwtSecretKey = etc.Get(defaultJWTSecretKey).String()

	if identityKey := etc.Get(defaultJWTIdentityKeyKey).String(); identityKey != "" {
		opts.jwtIdentityKey = identityKey
	}

	if signAlgorithm := etc.Get(defaultJWTSignAlgorithmKey).String(); signAlgorithm != "" {
		opts.jwtSignAlgorithm = xjwt.SignAlgorithm(signAlgorithm)
	}

	return opts
}

// WithID 设置实例ID
func WithID(id string) Option {
	return func(o *options) { o.id = id }
}

// WithName 设置实例名称
func WithName(name string) Option {
	return func(o *options) { o.name = name }
}

// WithAddr 设置监听地址
func WithAddr(addr string) Option {
	return func(o *options) { o.addr = addr }
}

// WithExpose 设置是否将内部通信地址暴露到公网
func WithExpose(expose bool) Option {
	return func(o *options) { o.expose = expose }
}

// WithContext 设置上下文
func WithContext(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

// WithServer 设置服务器
func WithServer(server network.Server) Option {
	return func(o *options) { o.server = server }
}

// WithTimeout 设置RPC调用超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(o *options) { o.timeout = timeout }
}

// WithLocator 设置用户定位器
func WithLocator(locator locate.Locator) Option {
	return func(o *options) { o.locator = locator }
}

// WithRegistry 设置服务注册器
func WithRegistry(r registry.Registry) Option {
	return func(o *options) { o.registry = r }
}

// WithDispatch 设置无状态路由消息分发策略
func WithDispatch(dispatch cluster.Dispatch) Option {
	return func(o *options) { o.dispatch = dispatch }
}

// WithMetadata 设置元数据
func WithMetadata(metadata map[string]string) Option {
	return func(o *options) {
		if len(metadata) != 0 {
			if len(o.metadata) == 0 {
				o.metadata = make(map[string]string)
			}

			maps.Copy(o.metadata, metadata)
		}
	}
}

func WithVersion(version string) Option {
	return func(o *options) { o.version = version }
}

func WithRetireDelay(delay time.Duration) Option {
	return func(o *options) { o.retireDelay = delay }
}

func WithReceiveQueue(size int) Option {
	return func(o *options) { o.receiveQueue = size }
}

func WithDeliverWorkers(n int) Option {
	return func(o *options) { o.deliverWorkers = n }
}

// WithLoginMessageID 设置 Gate 登录消息 ID；<=0 关闭 Gate 登录
func WithLoginMessageID(messageID int32) Option {
	return func(o *options) { o.loginMessageID = messageID }
}

// WithLobbyGameID 设置大厅 GameID，登录包与大厅绑定均使用该值
func WithLobbyGameID(gameID int32) Option {
	return func(o *options) { o.lobbyGameID = gameID }
}

// WithJWT 注入 JWT 解析器（HTTP 登录签发 token，Gate 校验并绑定用户）
func WithJWT(jwt *xjwt.JWT) Option {
	return func(o *options) { o.jwt = jwt }
}

// WithJWTSecretKey 设置 JWT 密钥（与 HTTP 服一致）
func WithJWTSecretKey(secretKey string) Option {
	return func(o *options) { o.jwtSecretKey = secretKey }
}

// WithJWTIdentityKey 设置 JWT 中用户 ID 字段名
func WithJWTIdentityKey(identityKey string) Option {
	return func(o *options) { o.jwtIdentityKey = identityKey }
}

// WithJWTSignAlgorithm 设置 JWT 签名算法
func WithJWTSignAlgorithm(signAlgorithm xjwt.SignAlgorithm) Option {
	return func(o *options) { o.jwtSignAlgorithm = signAlgorithm }
}
