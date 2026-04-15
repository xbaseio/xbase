package xants

import "time"

// Option 表示可选配置函数（函数式配置模式）。
type Option func(opts *Options)

// loadOptions 加载所有 Option，生成最终配置。
func loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}

// Options 定义协程池的所有配置项。
type Options struct {
	// ExpiryDuration 表示 worker 清理周期。
	// 后台清理协程会每隔该时间扫描一次，
	// 清理超过该时间未被使用的 worker。
	ExpiryDuration time.Duration

	// PreAlloc 是否在初始化时预分配 worker 内存。
	PreAlloc bool

	// MaxBlockingTasks 最大阻塞等待任务数。
	// 默认 0 表示不限制。
	MaxBlockingTasks int

	// Nonblocking 是否启用非阻塞模式。
	// 启用后 Submit 不会阻塞，直接返回错误（ErrPoolOverload）。
	// 启用后 MaxBlockingTasks 不生效。
	Nonblocking bool

	// PanicHandler worker 内 panic 的处理函数。
	// 如果为 nil，则使用默认行为（打印 panic 和堆栈）。
	PanicHandler func(any)

	// Logger 自定义日志接口。
	// 如果未设置，则使用标准 log。
	Logger Logger

	// DisablePurge 是否关闭 worker 自动清理。
	// 为 true 时，worker 常驻内存。
	DisablePurge bool
}

// WithOptions 直接设置完整 Options（整体覆盖）。
func WithOptions(options Options) Option {
	return func(opts *Options) {
		*opts = options
	}
}

// WithExpiryDuration 设置 worker 清理间隔。
func WithExpiryDuration(expiryDuration time.Duration) Option {
	return func(opts *Options) {
		opts.ExpiryDuration = expiryDuration
	}
}

// WithPreAlloc 设置是否预分配 worker。
func WithPreAlloc(preAlloc bool) Option {
	return func(opts *Options) {
		opts.PreAlloc = preAlloc
	}
}

// WithMaxBlockingTasks 设置最大阻塞任务数。
func WithMaxBlockingTasks(maxBlockingTasks int) Option {
	return func(opts *Options) {
		opts.MaxBlockingTasks = maxBlockingTasks
	}
}

// WithNonblocking 设置是否启用非阻塞模式。
func WithNonblocking(nonblocking bool) Option {
	return func(opts *Options) {
		opts.Nonblocking = nonblocking
	}
}

// WithPanicHandler 设置 panic 处理函数。
func WithPanicHandler(panicHandler func(any)) Option {
	return func(opts *Options) {
		opts.PanicHandler = panicHandler
	}
}

// WithLogger 设置自定义日志器。
func WithLogger(logger Logger) Option {
	return func(opts *Options) {
		opts.Logger = logger
	}
}

// WithDisablePurge 设置是否关闭自动清理 worker。
func WithDisablePurge(disable bool) Option {
	return func(opts *Options) {
		opts.DisablePurge = disable
	}
}
