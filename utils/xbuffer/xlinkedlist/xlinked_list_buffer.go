package xlinkedlist

import (
	"io"
	"math"

	bsPool "github.com/xbaseio/xbase/utils/xpool/xbyteslice"
)

// node 表示链表中的一个节点，内部持有一个 byte slice。
type node struct {
	buf  []byte
	next *node
}

// len 返回当前节点数据长度。
func (b *node) len() int {
	return len(b.buf)
}

// XBuffer 是一个基于链表实现的缓冲区。
type XBuffer struct {
	head  *node
	tail  *node
	size  int // 节点数量
	bytes int // 总字节数
}

// Read 从 XBuffer 中读取数据到 p。
func (llb *XBuffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	for b := llb.pop(); b != nil; b = llb.pop() {
		m := copy(p[n:], b.buf)
		n += m

		// 如果当前 node 没读完，把剩余部分重新放回头部
		if m < b.len() {
			b.buf = b.buf[m:]
			llb.pushFront(b)
		} else {
			bsPool.Put(b.buf)
		}

		if n == len(p) {
			return
		}
	}

	if n == 0 {
		err = io.EOF
	}
	return
}

// AllocNode 从池中分配一个指定长度的 []byte。
func (llb *XBuffer) AllocNode(n int) []byte {
	return bsPool.Get(n)
}

// FreeNode 将 []byte 放回池中释放。
func (llb *XBuffer) FreeNode(p []byte) {
	bsPool.Put(p)
}

// Append 类似 PushBack，但不会复制数据（直接挂引用）。
func (llb *XBuffer) Append(p []byte) {
	n := len(p)
	if n == 0 {
		return
	}
	llb.pushBack(&node{buf: p})
}

// Pop 移除并返回头节点的 buffer。
func (llb *XBuffer) Pop() []byte {
	n := llb.pop()
	if n == nil {
		return nil
	}
	return n.buf
}

// PushFront 在头部插入（会复制数据）。
func (llb *XBuffer) PushFront(p []byte) {
	n := len(p)
	if n == 0 {
		return
	}
	b := bsPool.Get(n)
	copy(b, p)
	llb.pushFront(&node{buf: b})
}

// PushBack 在尾部插入（会复制数据）。
func (llb *XBuffer) PushBack(p []byte) {
	n := len(p)
	if n == 0 {
		return
	}
	b := bsPool.Get(n)
	copy(b, p)
	llb.pushBack(&node{buf: b})
}

// Peek 返回最多 maxBytes 的数据（不会删除节点）。
func (llb *XBuffer) Peek(maxBytes int) ([][]byte, error) {
	if maxBytes <= 0 || maxBytes == math.MaxInt32 {
		maxBytes = math.MaxInt32
	} else if maxBytes > llb.Buffered() {
		return nil, io.ErrShortBuffer
	}

	var bs [][]byte
	var cum int

	for iter := llb.head; iter != nil; iter = iter.next {
		offset := iter.len()
		if cum+offset > maxBytes {
			offset = maxBytes - cum
		}
		bs = append(bs, iter.buf[:offset])

		if cum += offset; cum == maxBytes {
			break
		}
	}
	return bs, nil
}

// PeekWithBytes 类似 Peek，但会先拼接传入的 bs（通常来自 ring-buffer）。
func (llb *XBuffer) PeekWithBytes(maxBytes int, bs ...[]byte) ([][]byte, error) {
	if maxBytes <= 0 || maxBytes == math.MaxInt32 {
		maxBytes = math.MaxInt32
	} else if maxBytes > llb.Buffered() {
		return nil, io.ErrShortBuffer
	}

	var bss [][]byte
	var cum int

	// 先拼接外部数据
	for _, b := range bs {
		if n := len(b); n > 0 {
			offset := n
			if cum+offset > maxBytes {
				offset = maxBytes - cum
			}
			bss = append(bss, b[:offset])

			if cum += offset; cum == maxBytes {
				return bss, nil
			}
		}
	}

	// 再拼接链表数据
	for iter := llb.head; iter != nil; iter = iter.next {
		offset := iter.len()
		if cum+offset > maxBytes {
			offset = maxBytes - cum
		}
		bss = append(bss, iter.buf[:offset])

		if cum += offset; cum == maxBytes {
			break
		}
	}

	return bss, nil
}

// Discard 丢弃前 n 字节数据。
func (llb *XBuffer) Discard(n int) (discarded int, err error) {
	if n <= 0 {
		return
	}

	for n != 0 {
		b := llb.pop()
		if b == nil {
			break
		}

		if n < b.len() {
			b.buf = b.buf[n:]
			discarded += n
			llb.pushFront(b)
			break
		}

		n -= b.len()
		discarded += b.len()
		bsPool.Put(b.buf)
	}

	return
}

const minRead = 512

// ReadFrom 实现 io.ReaderFrom。
func (llb *XBuffer) ReadFrom(r io.Reader) (n int64, err error) {
	var m int
	for {
		b := bsPool.Get(minRead)
		m, err = r.Read(b)

		if m < 0 {
			panic("XBuffer.ReadFrom: reader returned negative count from Read")
		}

		n += int64(m)
		b = b[:m]

		if err == io.EOF {
			bsPool.Put(b)
			return n, nil
		}
		if err != nil {
			bsPool.Put(b)
			return
		}

		llb.pushBack(&node{buf: b})
	}
}

// WriteTo 实现 io.WriterTo。
func (llb *XBuffer) WriteTo(w io.Writer) (n int64, err error) {
	var m int
	for b := llb.pop(); b != nil; b = llb.pop() {
		m, err = w.Write(b.buf)

		if m > b.len() {
			panic("XBuffer.WriteTo: invalid Write count")
		}

		n += int64(m)

		if err != nil {
			return
		}

		if m < b.len() {
			b.buf = b.buf[m:]
			llb.pushFront(b)
			return n, io.ErrShortWrite
		}

		bsPool.Put(b.buf)
	}
	return
}

// Len 返回节点数量。
func (llb *XBuffer) Len() int {
	return llb.size
}

// Buffered 返回当前可读字节数。
func (llb *XBuffer) Buffered() int {
	return llb.bytes
}

// IsEmpty 判断是否为空。
func (llb *XBuffer) IsEmpty() bool {
	return llb.head == nil
}

// Reset 清空整个链表。
func (llb *XBuffer) Reset() {
	for b := llb.pop(); b != nil; b = llb.pop() {
		bsPool.Put(b.buf)
	}
	llb.head = nil
	llb.tail = nil
	llb.size = 0
	llb.bytes = 0
}

// pop 移除并返回头节点。
func (llb *XBuffer) pop() *node {
	if llb.head == nil {
		return nil
	}
	b := llb.head
	llb.head = b.next

	if llb.head == nil {
		llb.tail = nil
	}

	b.next = nil
	llb.size--
	llb.bytes -= b.len()
	return b
}

// pushFront 在头部插入节点。
func (llb *XBuffer) pushFront(b *node) {
	if b == nil {
		return
	}
	if llb.head == nil {
		b.next = nil
		llb.tail = b
	} else {
		b.next = llb.head
	}
	llb.head = b
	llb.size++
	llb.bytes += b.len()
}

// pushBack 在尾部插入节点。
func (llb *XBuffer) pushBack(b *node) {
	if b == nil {
		return
	}
	if llb.tail == nil {
		llb.head = b
	} else {
		llb.tail.next = b
	}
	b.next = nil
	llb.tail = b
	llb.size++
	llb.bytes += b.len()
}
