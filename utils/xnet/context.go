package xnet

import (
	"context"
	"net"
)

// contextKey 是 context.Context 中 Conn 值的键。
type contextKey struct{}

// NewContext 返回一个新的 context.Context，它携带将附加到 Conn 的值。
func NewContext(ctx context.Context, v any) context.Context {
	return context.WithValue(ctx, contextKey{}, v)
}

// FromContext 从 ctx 中检索存储的 Conn 的上下文值（如果有）。
func FromContext(ctx context.Context) any {
	return ctx.Value(contextKey{})
}

// connContextKey 是 context.Context 中 net.Conn 值的键。
type connContextKey struct{}

// NewNetConnContext 返回一个新的 context.Context，它携带 net.Conn 值。
func NewNetConnContext(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, connContextKey{}, c)
}

// FromNetConnContext 从 ctx 中检索 net.Conn 值（如果有）。
func FromNetConnContext(ctx context.Context) (net.Conn, bool) {
	c, ok := ctx.Value(connContextKey{}).(net.Conn)
	return c, ok
}

// netAddrContextKey 是 context.Context 中 net.Addr 值的键。
type netAddrContextKey struct{}

// NewNetAddrContext 返回一个新的 context.Context，它携带 net.Addr 值。
func NewNetAddrContext(ctx context.Context, a net.Addr) context.Context {
	return context.WithValue(ctx, netAddrContextKey{}, a)
}

// FromNetAddrContext 从 ctx 中检索 net.Addr 值（如果有）。
func FromNetAddrContext(ctx context.Context) (net.Addr, bool) {
	a, ok := ctx.Value(netAddrContextKey{}).(net.Addr)
	return a, ok
}
