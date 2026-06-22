package sse

import (
	"context"
	"net/http"
	"time"

	"github.com/xbaseio/xbase/etc"
)

const (
	defaultName              = "sse"
	defaultAddr              = ":8081"
	defaultPath              = "/events"
	defaultHealthPath        = "/healthz"
	defaultTopicQueryKey     = "topic"
	defaultClientBuffer      = 64
	defaultHeartbeatInterval = 15 * time.Second
	defaultShutdownTimeout   = 5 * time.Second
)

const (
	defaultNameKey              = "etc.sse.name"
	defaultAddrKey              = "etc.sse.addr"
	defaultPathKey              = "etc.sse.path"
	defaultHealthPathKey        = "etc.sse.healthPath"
	defaultTopicQueryKeyKey     = "etc.sse.topicQueryKey"
	defaultClientBufferKey      = "etc.sse.clientBuffer"
	defaultHeartbeatIntervalKey = "etc.sse.heartbeatInterval"
	defaultShutdownTimeoutKey   = "etc.sse.shutdownTimeout"
)

type ConnectHandler func(r *http.Request) (clientID string, topics []string, err error)

type Option func(o *options)

type options struct {
	ctx               context.Context
	name              string
	addr              string
	path              string
	healthPath        string
	topicQueryKey     string
	clientBuffer      int
	heartbeatInterval time.Duration
	shutdownTimeout   time.Duration
	connectHandler    ConnectHandler
}

func defaultOptions() *options {
	opts := &options{
		ctx:               context.Background(),
		name:              etc.Get(defaultNameKey, defaultName).String(),
		addr:              etc.Get(defaultAddrKey, defaultAddr).String(),
		path:              etc.Get(defaultPathKey, defaultPath).String(),
		healthPath:        etc.Get(defaultHealthPathKey, defaultHealthPath).String(),
		topicQueryKey:     etc.Get(defaultTopicQueryKeyKey, defaultTopicQueryKey).String(),
		clientBuffer:      etc.Get(defaultClientBufferKey, defaultClientBuffer).Int(),
		heartbeatInterval: etc.Get(defaultHeartbeatIntervalKey, defaultHeartbeatInterval.String()).Duration(),
		shutdownTimeout:   etc.Get(defaultShutdownTimeoutKey, defaultShutdownTimeout.String()).Duration(),
	}

	if opts.clientBuffer <= 0 {
		opts.clientBuffer = defaultClientBuffer
	}
	if opts.heartbeatInterval <= 0 {
		opts.heartbeatInterval = defaultHeartbeatInterval
	}
	if opts.shutdownTimeout <= 0 {
		opts.shutdownTimeout = defaultShutdownTimeout
	}
	if opts.path == "" {
		opts.path = defaultPath
	}
	if opts.healthPath == "" {
		opts.healthPath = defaultHealthPath
	}
	if opts.topicQueryKey == "" {
		opts.topicQueryKey = defaultTopicQueryKey
	}

	return opts
}

func WithContext(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

func WithName(name string) Option {
	return func(o *options) { o.name = name }
}

func WithAddr(addr string) Option {
	return func(o *options) { o.addr = addr }
}

func WithPath(path string) Option {
	return func(o *options) { o.path = path }
}

func WithHealthPath(path string) Option {
	return func(o *options) { o.healthPath = path }
}

func WithTopicQueryKey(key string) Option {
	return func(o *options) { o.topicQueryKey = key }
}

func WithClientBuffer(size int) Option {
	return func(o *options) {
		if size > 0 {
			o.clientBuffer = size
		}
	}
}

func WithHeartbeatInterval(interval time.Duration) Option {
	return func(o *options) {
		if interval > 0 {
			o.heartbeatInterval = interval
		}
	}
}

func WithShutdownTimeout(timeout time.Duration) Option {
	return func(o *options) {
		if timeout > 0 {
			o.shutdownTimeout = timeout
		}
	}
}

func WithConnectHandler(handler ConnectHandler) Option {
	return func(o *options) { o.connectHandler = handler }
}
