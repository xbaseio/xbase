/*
xGFD 结构：
| eventloop 索引 | conn matrix 行索引 | conn matrix 列索引 | 单调递增序列 | socket fd |
|    1 字节      |      1 字节        |       2 字节       |    4 字节    |   8 字节  |
*/
package xgfd

import (
	"encoding/binary"
	"math"
	"sync/atomic"
)

// xgfd 相关常量。
const (
	ConnMatrixColumnOffset = 2
	SequenceOffset         = 4
	FdOffset               = 8
	EventLoopIndexMax      = math.MaxUint8 + 1
	ConnMatrixRowMax       = math.MaxUint8 + 1
	ConnMatrixColumnMax    = math.MaxUint16 + 1
)

type monotoneSeq uint32

func (seq *monotoneSeq) Inc() uint32 {
	return atomic.AddUint32((*uint32)(seq), 1)
}

var monoSeq = new(monotoneSeq)

// GFD 用于存储 fd、eventloop 索引以及 connStore 索引信息。
type GFD [0x10]byte

// Fd 返回底层 fd。
func (gfd GFD) Fd() int {
	return int(binary.BigEndian.Uint64(gfd[FdOffset:]))
}

// EventLoopIndex 返回 eventloop 索引。
func (gfd GFD) EventLoopIndex() int {
	return int(gfd[0])
}

// ConnMatrixRow 返回 connMatrix 行索引。
func (gfd GFD) ConnMatrixRow() int {
	return int(gfd[1])
}

// ConnMatrixColumn 返回 connMatrix 列索引。
func (gfd GFD) ConnMatrixColumn() int {
	return int(binary.BigEndian.Uint16(gfd[ConnMatrixColumnOffset:SequenceOffset]))
}

// Sequence 返回单调递增序列，仅用于防止 fd 重复。
func (gfd GFD) Sequence() uint32 {
	return binary.BigEndian.Uint32(gfd[SequenceOffset:FdOffset])
}

// UpdateIndexes 更新 connStore 索引。
func (gfd *GFD) UpdateIndexes(row, column int) {
	(*gfd)[1] = byte(row)
	binary.BigEndian.PutUint16((*gfd)[ConnMatrixColumnOffset:SequenceOffset], uint16(column))
}

// Validate 检查 GFD 是否有效。
func (gfd GFD) Validate() bool {
	return gfd.Fd() > 2 && gfd.Fd() <= math.MaxInt &&
		gfd.EventLoopIndex() >= 0 && gfd.EventLoopIndex() < EventLoopIndexMax &&
		gfd.ConnMatrixRow() >= 0 && gfd.ConnMatrixRow() < ConnMatrixRowMax &&
		gfd.ConnMatrixColumn() >= 0 && gfd.ConnMatrixColumn() < ConnMatrixColumnMax &&
		gfd.Sequence() > 0
}

// NewGFD 创建一个新的 GFD。
func NewGFD(fd, elIndex, row, column int) (gfd GFD) {
	gfd[0] = byte(elIndex)
	gfd[1] = byte(row)
	binary.BigEndian.PutUint16(gfd[ConnMatrixColumnOffset:SequenceOffset], uint16(column))
	binary.BigEndian.PutUint32(gfd[SequenceOffset:FdOffset], monoSeq.Inc())
	binary.BigEndian.PutUint64(gfd[FdOffset:], uint64(fd))
	return
}
