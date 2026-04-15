package xelastic

import (
	"io"
	"math"

	"github.com/xbaseio/xbase/utils/xbuffer/xlinkedlist"
	errorx "github.com/xbaseio/xbase/xerrors"
)

// XBuffer 组合了 ring-buffer 和 list-buffer。
// ring-buffer 作为高优先级缓冲区用于存放响应数据；
// 只有当 ring-buffer 的数据量达到上限（maxStaticBytes）时，
// 才会切换到 list-buffer。
// list-buffer 更灵活、可扩展，有助于降低应用内存占用。
type XBuffer struct {
	maxStaticBytes int
	ringBuffer     XRingBuffer
	listBuffer     xlinkedlist.XBuffer
}

// New 创建并返回一个 elastic.XBuffer。
func New(maxStaticBytes int) (*XBuffer, error) {
	if maxStaticBytes <= 0 {
		return nil, errorx.ErrNegativeSize
	}
	return &XBuffer{maxStaticBytes: maxStaticBytes}, nil
}

// Read 从 XBuffer 中读取数据。
func (mb *XBuffer) Read(p []byte) (n int, err error) {
	n, err = mb.ringBuffer.Read(p)
	if n == len(p) {
		return n, err
	}

	var m int
	m, err = mb.listBuffer.Read(p[n:])
	n += m
	return
}

// Peek 以 [][]byte 的形式返回接下来的 n 个字节，
// 在调用 XBuffer.Discard() 之前，这些数据不会被丢弃。
func (mb *XBuffer) Peek(n int) ([][]byte, error) {
	if n <= 0 || n == math.MaxInt32 {
		n = math.MaxInt32
	} else if n > mb.Buffered() {
		return nil, io.ErrShortBuffer
	}

	head, tail := mb.ringBuffer.Peek(n)
	if mb.ringBuffer.Buffered() == n {
		return [][]byte{head, tail}, nil
	}
	return mb.listBuffer.PeekWithBytes(n, head, tail)
}

// Discard 丢弃当前缓冲区中的前 n 个字节。
func (mb *XBuffer) Discard(n int) (discarded int, err error) {
	discarded, err = mb.ringBuffer.Discard(n)
	if n <= discarded {
		return
	}

	n -= discarded
	var m int
	m, err = mb.listBuffer.Discard(n)
	discarded += m
	return
}

// Write 将数据追加到当前缓冲区。
func (mb *XBuffer) Write(p []byte) (n int, err error) {
	if !mb.listBuffer.IsEmpty() || mb.ringBuffer.Buffered() >= mb.maxStaticBytes {
		mb.listBuffer.PushBack(p)
		return len(p), nil
	}

	if mb.ringBuffer.Len() >= mb.maxStaticBytes {
		writable := mb.ringBuffer.Available()
		if n = len(p); n > writable {
			_, _ = mb.ringBuffer.Write(p[:writable])
			mb.listBuffer.PushBack(p[writable:])
			return
		}
	}

	return mb.ringBuffer.Write(p)
}

// Writev 将多个 byte 切片批量追加到当前缓冲区。
func (mb *XBuffer) Writev(bs [][]byte) (int, error) {
	if !mb.listBuffer.IsEmpty() || mb.ringBuffer.Buffered() >= mb.maxStaticBytes {
		var n int
		for _, b := range bs {
			mb.listBuffer.PushBack(b)
			n += len(b)
		}
		return n, nil
	}

	writable := mb.ringBuffer.Available()
	if mb.ringBuffer.Len() < mb.maxStaticBytes {
		writable = mb.maxStaticBytes - mb.ringBuffer.Buffered()
	}

	var pos, cum int
	for i, b := range bs {
		pos = i
		cum += len(b)
		if len(b) > writable {
			_, _ = mb.ringBuffer.Write(b[:writable])
			mb.listBuffer.PushBack(b[writable:])
			break
		}
		n, _ := mb.ringBuffer.Write(b)
		writable -= n
	}

	for pos++; pos < len(bs); pos++ {
		cum += len(bs[pos])
		mb.listBuffer.PushBack(bs[pos])
	}

	return cum, nil
}

// ReadFrom 实现 io.ReaderFrom 接口。
func (mb *XBuffer) ReadFrom(r io.Reader) (int64, error) {
	if !mb.listBuffer.IsEmpty() || mb.ringBuffer.Buffered() >= mb.maxStaticBytes {
		return mb.listBuffer.ReadFrom(r)
	}
	return mb.ringBuffer.ReadFrom(r)
}

// WriteTo 实现 io.WriterTo 接口。
func (mb *XBuffer) WriteTo(w io.Writer) (n int64, err error) {
	if n, err = mb.ringBuffer.WriteTo(w); err != nil {
		return
	}

	var m int64
	m, err = mb.listBuffer.WriteTo(w)
	n += m
	return
}

// Buffered 返回当前缓冲区中可读取的字节总数。
func (mb *XBuffer) Buffered() int {
	return mb.ringBuffer.Buffered() + mb.listBuffer.Buffered()
}

// IsEmpty 判断当前缓冲区是否为空。
func (mb *XBuffer) IsEmpty() bool {
	return mb.ringBuffer.IsEmpty() && mb.listBuffer.IsEmpty()
}

// Reset 重置缓冲区状态。
func (mb *XBuffer) Reset(maxStaticBytes int) {
	mb.ringBuffer.Reset()
	mb.listBuffer.Reset()
	if maxStaticBytes > 0 {
		mb.maxStaticBytes = maxStaticBytes
	}
}

// Release 释放该缓冲区占用的所有资源。
func (mb *XBuffer) Release() {
	mb.ringBuffer.Done()
	mb.listBuffer.Reset()
}
