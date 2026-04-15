package xnet

import (
	"time"
)

//
// =========================
// Option 基础
// =========================
//

// Option 是一个用于修改 Options 的函数。
type Option func(opts *Options)

// loadOptions 加载所有 Option。
func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}

//
// =========================
// TCP 选项
// =========================
//

// TCPSocketOpt 表示 TCP 套接字行为。
type TCPSocketOpt int

const (
	// TCPNoDelay 禁用 Nagle（低延迟）
	TCPNoDelay TCPSocketOpt = iota

	// TCPDelay 启用 Nagle（减少小包）
	TCPDelay
)

//
// =========================
// 核心配置
// =========================
//

// Options 是 xnet 的全局配置。
type Options struct {

	// =========================
	// 负载均衡
	// =========================

	// LB 连接分发策略（仅服务端生效）
	LB LoadBalancing

	// =========================
	// Socket基础
	// =========================

	ReuseAddr bool // SO_REUSEADDR
	ReusePort bool // SO_REUSEPORT

	MulticastInterfaceIndex int    // UDP组播网卡索引
	BindToDevice            string // 绑定网卡（Linux）

	// =========================
	// 并发模型
	// =========================

	// Multicore 是否使用多核（自动CPU数）
	Multicore bool

	// NumEventLoop 手动指定 loop 数（优先级更高）
	NumEventLoop int

	// LockOSThread 绑定 OS 线程（用于 cgo / TLS / GL）
	LockOSThread bool

	// =========================
	// Buffer
	// =========================

	// 读缓冲（默认 64KB，自动转2次幂）
	ReadBufferCap int

	// 写缓冲（默认 64KB，自动转2次幂）
	WriteBufferCap int

	// =========================
	// TCP
	// =========================

	// TCP keepalive
	TCPKeepAlive    time.Duration
	TCPKeepInterval time.Duration
	TCPKeepCount    int

	// TCP_NODELAY
	TCPNoDelay TCPSocketOpt

	// =========================
	// Socket Buffer
	// =========================

	SocketRecvBuffer int
	SocketSendBuffer int

	// =========================
	// Loop行为
	// =========================

	Ticker bool // 是否开启 OnTick

	// =========================
	// 高级（ET模式）
	// =========================

	// 是否开启边缘触发
	EdgeTriggeredIO bool

	// 单次最大读写（ET防饿死）
	EdgeTriggeredIOChunk int
}

//
// =========================
// 全量覆盖
// =========================
//

// WithOptions 覆盖所有配置。
func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

//
// =========================
// 并发
// =========================
//

func WithMulticore(multicore bool) Option {
	return func(opts *Options) {
		opts.Multicore = multicore
	}
}

func WithNumEventLoop(numEventLoop int) Option {
	return func(opts *Options) {
		opts.NumEventLoop = numEventLoop
	}
}

func WithLockOSThread(lockOSThread bool) Option {
	return func(opts *Options) {
		opts.LockOSThread = lockOSThread
	}
}

//
// =========================
// Buffer
// =========================
//

func WithReadBufferCap(n int) Option {
	return func(opts *Options) {
		opts.ReadBufferCap = n
	}
}

func WithWriteBufferCap(n int) Option {
	return func(opts *Options) {
		opts.WriteBufferCap = n
	}
}

//
// =========================
// LoadBalance
// =========================
//

func WithLoadBalancing(lb LoadBalancing) Option {
	return func(opts *Options) {
		opts.LB = lb
	}
}

//
// =========================
// Socket
// =========================
//

func WithReusePort(b bool) Option {
	return func(opts *Options) {
		opts.ReusePort = b
	}
}

func WithReuseAddr(b bool) Option {
	return func(opts *Options) {
		opts.ReuseAddr = b
	}
}

func WithSocketRecvBuffer(n int) Option {
	return func(opts *Options) {
		opts.SocketRecvBuffer = n
	}
}

func WithSocketSendBuffer(n int) Option {
	return func(opts *Options) {
		opts.SocketSendBuffer = n
	}
}

//
// =========================
// TCP
// =========================
//

func WithTCPKeepAlive(d time.Duration) Option {
	return func(opts *Options) {
		opts.TCPKeepAlive = d
	}
}

func WithTCPKeepInterval(d time.Duration) Option {
	return func(opts *Options) {
		opts.TCPKeepInterval = d
	}
}

func WithTCPKeepCount(n int) Option {
	return func(opts *Options) {
		opts.TCPKeepCount = n
	}
}

func WithTCPNoDelay(v TCPSocketOpt) Option {
	return func(opts *Options) {
		opts.TCPNoDelay = v
	}
}

//
// =========================
// 业务增强
// =========================
//

func WithTicker(b bool) Option {
	return func(opts *Options) {
		opts.Ticker = b
	}
}

func WithMulticastInterfaceIndex(idx int) Option {
	return func(opts *Options) {
		opts.MulticastInterfaceIndex = idx
	}
}

func WithBindToDevice(iface string) Option {
	return func(opts *Options) {
		opts.BindToDevice = iface
	}
}

//
// =========================
// 高级（ET模式）
// =========================
//

func WithEdgeTriggeredIO(et bool) Option {
	return func(opts *Options) {
		opts.EdgeTriggeredIO = et
	}
}

func WithEdgeTriggeredIOChunk(chunk int) Option {
	return func(opts *Options) {
		opts.EdgeTriggeredIOChunk = chunk
	}
}
