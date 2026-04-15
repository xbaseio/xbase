// Package xbytebuffer 提供 bytebufferpool 的轻量封装（对象复用，降低 GC）。
package xbytebuffer

import "github.com/valyala/bytebufferpool"

// ByteBuffer 是 bytebufferpool.ByteBuffer 的类型别名。
type ByteBuffer = bytebufferpool.ByteBuffer

// Get 从对象池获取一个空的 ByteBuffer。
var Get = bytebufferpool.Get

// Put 将 ByteBuffer 放回对象池。
// 调用方需保证 b != nil（避免热路径分支判断）。
var Put = func(b *ByteBuffer) {
	if b != nil {
		bytebufferpool.Put(b)
	}
}
