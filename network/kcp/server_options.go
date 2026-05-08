package kcp

import (
	"time"

	"github.com/xbaseio/xbase/etc"
)

const (
	defaultServerAddr             = ":3553"
	defaultServerMaxConnNum       = 5000
	defaultServerAuthorizeTimeout = "0s"
)

const (
	defaultServerAddrKey             = "etc.network.kcp.server.addr"
	defaultServerMaxConnNumKey       = "etc.network.kcp.server.maxConnNum"
	defaultServerAuthorizeTimeoutKey = "etc.network.kcp.server.authorizeTimeout"
	defaultServerMtuKey              = "etc.network.kcp.server.mtu"
	defaultServerNoDelayKey          = "etc.network.kcp.server.noDelay"
	defaultServerAckNoDelayKey       = "etc.network.kcp.server.ackNoDelay"
	defaultServerWriteDelayKey       = "etc.network.kcp.server.writeDelay"
	defaultServerWindowSizeKey       = "etc.network.kcp.server.windowSize"
	defaultServerReadBufferKey       = "etc.network.kcp.server.readBuffer"
	defaultServerWriteBufferKey      = "etc.network.kcp.server.writeBuffer"
)

type ServerOption func(o *serverOptions)

type serverOptions struct {
	addr             string        // 监听地址
	maxConnNum       int           // 最大连接数
	authorizeTimeout time.Duration // 授权超时时间，默认0s，不检测
	mtu              int           // 最大传输单元，默认不设置
	noDelay          []int         // 是否开启无延迟模式，默认不设置
	ackNoDelay       bool          // 是否开启ACK延迟确认，默认不设置
	writeDelay       bool          // 是否开启写延迟，默认不设置
	windowSize       []int         // 窗口大小，默认不设置
	readBuffer       int           // 读取缓冲区大小，默认不设置
	writeBuffer      int           // 写入缓冲区大小，默认不设置
}

func defaultServerOptions() *serverOptions {
	return &serverOptions{
		addr:             etc.Get(defaultServerAddrKey, defaultServerAddr).String(),
		maxConnNum:       etc.Get(defaultServerMaxConnNumKey, defaultServerMaxConnNum).Int(),
		authorizeTimeout: etc.Get(defaultServerAuthorizeTimeoutKey, defaultServerAuthorizeTimeout).Duration(),
		mtu:              etc.Get(defaultServerMtuKey).Int(),
		noDelay:          etc.Get(defaultServerNoDelayKey).Ints(),
		ackNoDelay:       etc.Get(defaultServerAckNoDelayKey).Bool(),
		writeDelay:       etc.Get(defaultServerWriteDelayKey).Bool(),
		windowSize:       etc.Get(defaultServerWindowSizeKey).Ints(),
		readBuffer:       int(etc.Get(defaultServerReadBufferKey).B()),
		writeBuffer:      int(etc.Get(defaultServerWriteBufferKey).B()),
	}
}

// WithServerListenAddr 设置监听地址
func WithServerListenAddr(addr string) ServerOption {
	return func(o *serverOptions) { o.addr = addr }
}

// WithServerMaxConnNum 设置连接的最大连接数
func WithServerMaxConnNum(maxConnNum int) ServerOption {
	return func(o *serverOptions) { o.maxConnNum = maxConnNum }
}

// WithServerAuthorizeTimeout 设置授权超时时间
func WithServerAuthorizeTimeout(authorizeTimeout time.Duration) ServerOption {
	return func(o *serverOptions) { o.authorizeTimeout = authorizeTimeout }
}

// WithServerMtu 设置最大传输单元
func WithServerMtu(mtu int) ServerOption {
	return func(o *serverOptions) { o.mtu = mtu }
}

// WithServerNoDelay 设置是否开启无延迟模式
func WithServerNoDelay(noDelay []int) ServerOption {
	return func(o *serverOptions) { o.noDelay = noDelay }
}

// WithServerAckNoDelay 设置是否开启ACK延迟确认
func WithServerAckNoDelay(ackNoDelay bool) ServerOption {
	return func(o *serverOptions) { o.ackNoDelay = ackNoDelay }
}

// WithServerWriteDelay 设置是否开启写延迟
func WithServerWriteDelay(writeDelay bool) ServerOption {
	return func(o *serverOptions) { o.writeDelay = writeDelay }
}

// WithServerWindowSize 设置窗口大小
func WithServerWindowSize(windowSize []int) ServerOption {
	return func(o *serverOptions) { o.windowSize = windowSize }
}

// WithServerReadBuffer 设置读取缓冲区大小
func WithServerReadBuffer(readBuffer int) ServerOption {
	return func(o *serverOptions) { o.readBuffer = readBuffer }
}

// WithServerWriteBuffer 设置写入缓冲区大小
func WithServerWriteBuffer(writeBuffer int) ServerOption {
	return func(o *serverOptions) { o.writeBuffer = writeBuffer }
}
