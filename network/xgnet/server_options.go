package xgnet

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
	defaultServerAddrKey             = "etc.network.tcp.server.addr"
	defaultServerCertFileKey         = "etc.network.tcp.server.certFile"
	defaultServerKeyFileKey          = "etc.network.tcp.server.keyFile"
	defaultServerMaxConnNumKey       = "etc.network.tcp.server.maxConnNum"
	defaultServerAuthorizeTimeoutKey = "etc.network.tcp.server.authorizeTimeout"
)

type ServerOption func(o *serverOptions)

type serverOptions struct {
	addr             string        // 监听地址，默认0.0.0.0:3553
	certFile         string        // 证书文件
	keyFile          string        // 秘钥文件
	maxConnNum       int           // 最大连接数，默认5000
	authorizeTimeout time.Duration // 授权超时时间，默认0s，不检测
}

func defaultServerOptions() *serverOptions {
	return &serverOptions{
		addr:             etc.Get(defaultServerAddrKey, defaultServerAddr).String(),
		certFile:         etc.Get(defaultServerCertFileKey).String(),
		keyFile:          etc.Get(defaultServerKeyFileKey).String(),
		maxConnNum:       etc.Get(defaultServerMaxConnNumKey, defaultServerMaxConnNum).Int(),
		authorizeTimeout: etc.Get(defaultServerAuthorizeTimeoutKey, defaultServerAuthorizeTimeout).Duration(),
	}
}

// WithServerListenAddr 设置监听地址
func WithServerListenAddr(addr string) ServerOption {
	return func(o *serverOptions) { o.addr = addr }
}

// WithServerCredentials 设置服务器证书和秘钥
func WithServerCredentials(certFile, keyFile string) ServerOption {
	return func(o *serverOptions) { o.certFile, o.keyFile = certFile, keyFile }
}

// WithServerMaxConnNum 设置连接的最大连接数
func WithServerMaxConnNum(maxConnNum int) ServerOption {
	return func(o *serverOptions) { o.maxConnNum = maxConnNum }
}

// WithServerAuthorizeTimeout 设置授权超时时间
func WithServerAuthorizeTimeout(authorizeTimeout time.Duration) ServerOption {
	return func(o *serverOptions) { o.authorizeTimeout = authorizeTimeout }
}
