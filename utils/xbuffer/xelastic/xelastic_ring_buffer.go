package xelastic

import (
	"io"

	"github.com/xbaseio/xbase/utils/xbuffer/xring"
	rbPool "github.com/xbaseio/xbase/utils/xpool/xringbuffer"
)

// XRingBuffer 是 xring.XBuffer 的弹性封装。
type XRingBuffer struct {
	rb *xring.XBuffer
}

func (b *XRingBuffer) instance() *xring.XBuffer {
	if b.rb == nil {
		b.rb = rbPool.Get()
	}

	return b.rb
}

// Done 检查并将内部 xring-buffer 归还到对象池。
func (b *XRingBuffer) Done() {
	if b.rb != nil {
		rbPool.Put(b.rb)
		b.rb = nil
	}
}

func (b *XRingBuffer) done() {
	if b.rb != nil && b.rb.IsEmpty() {
		rbPool.Put(b.rb)
		b.rb = nil
	}
}

// Peek 返回接下来的 n 个字节，但不会推进读指针。
// 当 n <= 0 时，返回全部可读数据。
func (b *XRingBuffer) Peek(n int) (head []byte, tail []byte) {
	if b.rb == nil {
		return nil, nil
	}
	return b.rb.Peek(n)
}

// Discard 通过推进读指针，跳过接下来的 n 个字节。
func (b *XRingBuffer) Discard(n int) (int, error) {
	if b.rb == nil {
		return 0, xring.ErrIsEmpty
	}

	defer b.done()
	return b.rb.Discard(n)
}

// Read 读取最多 len(p) 个字节到 p 中。
// 返回实际读取的字节数（0 <= n <= len(p)）以及读取过程中遇到的错误。
//
// 即使 Read 返回 n < len(p)，实现内部仍可能把整个 p 当作临时缓冲区使用。
// 如果当前可读数据不足 len(p)，Read 会按照惯例返回当前可读的内容，而不是等待更多数据。
// 如果 Read 在成功读取 n > 0 个字节后遇到错误或 EOF，
// 会先返回已读取的字节数，并可能同时返回该错误；
// 也可能在下一次调用中返回该错误（并且 n == 0）。
// 调用方应始终优先处理 n > 0 的数据，再处理 err。
func (b *XRingBuffer) Read(p []byte) (int, error) {
	if b.rb == nil {
		return 0, xring.ErrIsEmpty
	}

	defer b.done()
	return b.rb.Read(p)
}

// ReadByte 读取并返回下一个字节；如果为空则返回 ErrIsEmpty。
func (b *XRingBuffer) ReadByte() (byte, error) {
	if b.rb == nil {
		return 0, xring.ErrIsEmpty
	}

	defer b.done()
	return b.rb.ReadByte()
}

// Write 将 p 的内容写入底层缓冲区。
// 返回实际写入的字节数（n == len(p)）以及可能的错误。
//
// 如果 p 的长度大于当前 xring-buffer 的可写容量，
// 会自动为该 xring-buffer 扩容。
// Write 在任何情况下都不能修改传入的切片内容。
func (b *XRingBuffer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return b.instance().Write(p)
}

// WriteByte 向缓冲区写入一个字节。
func (b *XRingBuffer) WriteByte(c byte) error {
	return b.instance().WriteByte(c)
}

// Buffered 返回当前可读字节数。
func (b *XRingBuffer) Buffered() int {
	if b.rb == nil {
		return 0
	}
	return b.rb.Buffered()
}

// Len 返回底层缓冲区长度。
func (b *XRingBuffer) Len() int {
	if b.rb == nil {
		return 0
	}
	return b.rb.Len()
}

// Cap 返回底层缓冲区容量。
func (b *XRingBuffer) Cap() int {
	if b.rb == nil {
		return 0
	}
	return b.rb.Cap()
}

// Available 返回当前可写字节数。
func (b *XRingBuffer) Available() int {
	if b.rb == nil {
		return 0
	}
	return b.rb.Available()
}

// WriteString 将字符串 s 写入缓冲区。
func (b *XRingBuffer) WriteString(s string) (int, error) {
	if len(s) == 0 {
		return 0, nil
	}
	return b.instance().WriteString(s)
}

// Bytes 返回全部当前可读数据。
// 它不会移动读指针，但会复制可读数据。
func (b *XRingBuffer) Bytes() []byte {
	if b.rb == nil {
		return nil
	}
	return b.rb.Bytes()
}

// ReadFrom 实现 io.ReaderFrom 接口。
func (b *XRingBuffer) ReadFrom(r io.Reader) (int64, error) {
	return b.instance().ReadFrom(r)
}

// WriteTo 实现 io.WriterTo 接口。
func (b *XRingBuffer) WriteTo(w io.Writer) (int64, error) {
	if b.rb == nil {
		return 0, xring.ErrIsEmpty
	}

	defer b.done()
	return b.instance().WriteTo(w)
}

// IsFull 判断当前 xring-buffer 是否已满。
func (b *XRingBuffer) IsFull() bool {
	if b.rb == nil {
		return false
	}
	return b.rb.IsFull()
}

// IsEmpty 判断当前 xring-buffer 是否为空。
func (b *XRingBuffer) IsEmpty() bool {
	if b.rb == nil {
		return true
	}
	return b.rb.IsEmpty()
}

// Reset 将读指针和写指针重置为 0。
func (b *XRingBuffer) Reset() {
	if b.rb == nil {
		return
	}
	b.rb.Reset()
}
