package xring

import (
	"errors"
	"io"

	"github.com/xbaseio/xbase/utils/xbs"
	"github.com/xbaseio/xbase/utils/xmath"
	bsPool "github.com/xbaseio/xbase/utils/xpool/xbyteslice"
)

const (
	// MinRead 是 XBuffer.ReadFrom 调用 Read 时使用的最小缓冲区大小。
	// 只要 XBuffer 除了承载 r 的数据外还有至少 MinRead 的剩余空间，
	// ReadFrom 就不会触发底层 buffer 扩容。
	MinRead = 512

	// DefaultBufferSize 是 xring-buffer 首次分配的默认大小。
	DefaultBufferSize   = 1024     // 1KB
	bufferGrowThreshold = 4 * 1024 // 4KB
)

// ErrIsEmpty 当尝试从空的 xring-buffer 读取时返回该错误。
var ErrIsEmpty = errors.New("xring-buffer is empty")

// XBuffer 是一个环形缓冲区，实现了 io.Reader 和 io.Writer 接口。
type XBuffer struct {
	buf     []byte
	size    int
	r       int // 下一次读取的位置
	w       int // 下一次写入的位置
	isEmpty bool
}

// New 创建一个新的 XBuffer，并指定初始容量。
func New(size int) *XBuffer {
	if size == 0 {
		return &XBuffer{isEmpty: true}
	}
	size = xmath.CeilToPowerOfTwo(size)
	return &XBuffer{
		buf:     make([]byte, size),
		size:    size,
		isEmpty: true,
	}
}

// Peek 返回接下来的 n 个字节，但不会推进读指针。
// 当 n <= 0 时返回全部数据。
func (rb *XBuffer) Peek(n int) (head []byte, tail []byte) {
	if rb.isEmpty {
		return
	}

	if n <= 0 {
		return rb.peekAll()
	}

	if rb.w > rb.r {
		m := rb.w - rb.r // 当前连续可读数据长度
		if m > n {
			m = n
		}
		head = rb.buf[rb.r : rb.r+m]
		return
	}

	m := rb.size - rb.r + rb.w // 当前总可读数据长度
	if m > n {
		m = n
	}

	if rb.r+m <= rb.size {
		head = rb.buf[rb.r : rb.r+m]
	} else {
		c1 := rb.size - rb.r
		head = rb.buf[rb.r:]
		c2 := m - c1
		tail = rb.buf[:c2]
	}

	return
}

// peekAll 返回全部数据（不移动读指针）。
func (rb *XBuffer) peekAll() (head []byte, tail []byte) {
	if rb.isEmpty {
		return
	}

	if rb.w > rb.r {
		head = rb.buf[rb.r:rb.w]
		return
	}

	head = rb.buf[rb.r:]
	if rb.w != 0 {
		tail = rb.buf[:rb.w]
	}

	return
}

// Discard 跳过接下来的 n 个字节（推进读指针）。
func (rb *XBuffer) Discard(n int) (discarded int, err error) {
	if n <= 0 {
		return 0, nil
	}

	discarded = rb.Buffered()
	if n < discarded {
		rb.r = (rb.r + n) % rb.size
		return n, nil
	}
	rb.Reset()
	return
}

// Read 从缓冲区读取最多 len(p) 个字节到 p 中。
// 返回读取的字节数以及可能的错误，行为符合 io.Reader 规范。
func (rb *XBuffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	if rb.isEmpty {
		return 0, ErrIsEmpty
	}

	if rb.w > rb.r {
		n = rb.w - rb.r
		if n > len(p) {
			n = len(p)
		}
		copy(p, rb.buf[rb.r:rb.r+n])
		rb.r += n
		if rb.r == rb.w {
			rb.Reset()
		}
		return
	}

	n = rb.size - rb.r + rb.w
	if n > len(p) {
		n = len(p)
	}

	if rb.r+n <= rb.size {
		copy(p, rb.buf[rb.r:rb.r+n])
	} else {
		c1 := rb.size - rb.r
		copy(p, rb.buf[rb.r:])
		c2 := n - c1
		copy(p[c1:], rb.buf[:c2])
	}
	rb.r = (rb.r + n) % rb.size
	if rb.r == rb.w {
		rb.Reset()
	}

	return
}

// ReadByte 读取并返回下一个字节；若为空则返回 ErrIsEmpty。
func (rb *XBuffer) ReadByte() (b byte, err error) {
	if rb.isEmpty {
		return 0, ErrIsEmpty
	}
	b = rb.buf[rb.r]
	rb.r++
	if rb.r == rb.size {
		rb.r = 0
	}
	if rb.r == rb.w {
		rb.Reset()
	}

	return
}

// Write 将 p 的内容写入缓冲区。
// 如果空间不足，会自动扩容。
func (rb *XBuffer) Write(p []byte) (n int, err error) {
	n = len(p)
	if n == 0 {
		return
	}

	free := rb.Available()
	if n > free {
		rb.grow(rb.size + n - free)
	}

	if rb.w >= rb.r {
		c1 := rb.size - rb.w
		if c1 >= n {
			copy(rb.buf[rb.w:], p)
			rb.w += n
		} else {
			copy(rb.buf[rb.w:], p[:c1])
			c2 := n - c1
			copy(rb.buf, p[c1:])
			rb.w = c2
		}
	} else {
		copy(rb.buf[rb.w:], p)
		rb.w += n
	}

	if rb.w == rb.size {
		rb.w = 0
	}

	rb.isEmpty = false

	return
}

// WriteByte 向缓冲区写入一个字节。
func (rb *XBuffer) WriteByte(c byte) error {
	if rb.Available() < 1 {
		rb.grow(1)
	}
	rb.buf[rb.w] = c
	rb.w++

	if rb.w == rb.size {
		rb.w = 0
	}
	rb.isEmpty = false

	return nil
}

// Buffered 返回当前可读数据长度。
func (rb *XBuffer) Buffered() int {
	if rb.r == rb.w {
		if rb.isEmpty {
			return 0
		}
		return rb.size
	}

	if rb.w > rb.r {
		return rb.w - rb.r
	}

	return rb.size - rb.r + rb.w
}

// Len 返回底层切片长度。
func (rb *XBuffer) Len() int {
	return len(rb.buf)
}

// Cap 返回缓冲区容量。
func (rb *XBuffer) Cap() int {
	return rb.size
}

// Available 返回当前可写空间大小。
func (rb *XBuffer) Available() int {
	if rb.r == rb.w {
		if rb.isEmpty {
			return rb.size
		}
		return 0
	}

	if rb.w < rb.r {
		return rb.r - rb.w
	}

	return rb.size - rb.w + rb.r
}

// WriteString 将字符串写入缓冲区。
func (rb *XBuffer) WriteString(s string) (int, error) {
	return rb.Write(xbs.StringToBytes(s))
}

// Bytes 返回当前全部可读数据。
// 该方法不会移动读指针，但会复制可读数据。
func (rb *XBuffer) Bytes() []byte {
	if rb.isEmpty {
		return nil
	} else if rb.w == rb.r {
		var bb []byte
		bb = append(bb, rb.buf[rb.r:]...)
		bb = append(bb, rb.buf[:rb.w]...)
		return bb
	}

	var bb []byte
	if rb.w > rb.r {
		bb = append(bb, rb.buf[rb.r:rb.w]...)
		return bb
	}

	bb = append(bb, rb.buf[rb.r:]...)

	if rb.w != 0 {
		bb = append(bb, rb.buf[:rb.w]...)
	}

	return bb
}

// ReadFrom 实现 io.ReaderFrom 接口。
func (rb *XBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	var m int
	for {
		if rb.Available() < MinRead {
			rb.grow(rb.Buffered() + MinRead)
		}

		if rb.w >= rb.r {
			m, err = r.Read(rb.buf[rb.w:])
			if m < 0 {
				panic("RingBuffer.ReadFrom: reader returned negative count from Read")
			}
			rb.isEmpty = false
			rb.w = (rb.w + m) % rb.size
			n += int64(m)
			if err == io.EOF {
				return n, nil
			}
			if err != nil {
				return
			}

			m, err = r.Read(rb.buf[:rb.r])
			if m < 0 {
				panic("RingBuffer.ReadFrom: reader returned negative count from Read")
			}
			rb.w = (rb.w + m) % rb.size
			n += int64(m)
			if err == io.EOF {
				return n, nil
			}
			if err != nil {
				return
			}
		} else {
			m, err = r.Read(rb.buf[rb.w:rb.r])
			if m < 0 {
				panic("RingBuffer.ReadFrom: reader returned negative count from Read")
			}
			rb.isEmpty = false
			rb.w = (rb.w + m) % rb.size
			n += int64(m)
			if err == io.EOF {
				return n, nil
			}
			if err != nil {
				return
			}
		}
	}
}

// WriteTo 实现 io.WriterTo 接口。
func (rb *XBuffer) WriteTo(w io.Writer) (int64, error) {
	if rb.isEmpty {
		return 0, ErrIsEmpty
	}

	if rb.w > rb.r {
		n := rb.w - rb.r
		m, err := w.Write(rb.buf[rb.r : rb.r+n])
		if m > n {
			panic("RingBuffer.WriteTo: invalid Write count")
		}
		rb.r += m
		if rb.r == rb.w {
			rb.Reset()
		}
		if err != nil {
			return int64(m), err
		}
		if !rb.isEmpty {
			return int64(m), io.ErrShortWrite
		}
		return int64(m), nil
	}

	n := rb.size - rb.r + rb.w
	if rb.r+n <= rb.size {
		m, err := w.Write(rb.buf[rb.r : rb.r+n])
		if m > n {
			panic("RingBuffer.WriteTo: invalid Write count")
		}
		rb.r = (rb.r + m) % rb.size
		if rb.r == rb.w {
			rb.Reset()
		}
		if err != nil {
			return int64(m), err
		}
		if !rb.isEmpty {
			return int64(m), io.ErrShortWrite
		}
		return int64(m), nil
	}

	var cum int64
	c1 := rb.size - rb.r
	m, err := w.Write(rb.buf[rb.r:])
	if m > c1 {
		panic("RingBuffer.WriteTo: invalid Write count")
	}
	rb.r = (rb.r + m) % rb.size
	if err != nil {
		return int64(m), err
	}
	if m < c1 {
		return int64(m), io.ErrShortWrite
	}
	cum += int64(m)

	c2 := n - c1
	m, err = w.Write(rb.buf[:c2])
	if m > c2 {
		panic("RingBuffer.WriteTo: invalid Write count")
	}
	rb.r = m
	cum += int64(m)
	if rb.r == rb.w {
		rb.Reset()
	}
	if err != nil {
		return cum, err
	}
	if !rb.isEmpty {
		return cum, io.ErrShortWrite
	}
	return cum, nil
}

// IsFull 判断当前 xring-buffer 是否已满。
func (rb *XBuffer) IsFull() bool {
	return rb.r == rb.w && !rb.isEmpty
}

// IsEmpty 判断当前 xring-buffer 是否为空。
func (rb *XBuffer) IsEmpty() bool {
	return rb.isEmpty
}

// Reset 将读指针和写指针重置为 0。
func (rb *XBuffer) Reset() {
	rb.isEmpty = true
	rb.r, rb.w = 0, 0
}

func (rb *XBuffer) grow(newCap int) {
	if n := rb.size; n == 0 {
		if newCap <= DefaultBufferSize {
			newCap = DefaultBufferSize
		} else {
			newCap = xmath.CeilToPowerOfTwo(newCap)
		}
	} else {
		doubleCap := n + n
		if newCap <= doubleCap {
			if n < bufferGrowThreshold {
				newCap = doubleCap
			} else {
				// 检查 0 < n，用于检测溢出并避免死循环。
				for 0 < n && n < newCap {
					n += n / 4
				}
				// 如果 n 计算未溢出，则使用 n 作为新容量。
				if n > 0 {
					newCap = n
				}
			}
		}
	}

	newBuf := bsPool.Get(newCap)
	oldLen := rb.Buffered()
	_, _ = rb.Read(newBuf)
	bsPool.Put(rb.buf)
	rb.buf = newBuf
	rb.r = 0
	rb.w = oldLen
	rb.size = newCap
	if rb.w > 0 {
		rb.isEmpty = false
	}
}
