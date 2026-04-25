package packet

import (
	"encoding/binary"
	"strings"

	"github.com/xbaseio/xbase/etc"
)

// heartbeat packet
// ------------------------------------------------------------------------------
// | size(4 byte) = (1 byte + 8 byte) | header(1 byte) | heartbeat time(8 byte) |
// ------------------------------------------------------------------------------

// data packet
// -----------------------------------------------------------------------------------------------------------------------
// | size(4 byte) = (1 byte + n byte + m byte + x byte) | header(1 byte) | route(n byte) | seq(m byte) | message(x byte) |
// -----------------------------------------------------------------------------------------------------------------------

const (
	littleEndian = "little"
	bigEndian    = "big"
)

const (
	defaultSizeBytes          = 4
	defaultHeaderBytes        = 4
	defaultNodeIDBytes        = 4
	defaultMessageIDBytes     = 4
	defaultSeqBytes           = 4
	defaultBufferBytes        = 5000
	defaultHeartbeatTime      = false
	defaultHeartbeatTimeBytes = 8
	defaultHeaderSize         = 16
)

const (
	defaultEndianKey        = "etc.packet.byteOrder"
	defaultBufferBytesKey   = "etc.packet.bufferBytes"
	defaultHeartbeatTimeKey = "etc.packet.heartbeatTime"
)

type options struct {
	// 字节序
	// 默认为binary.LittleEndian
	byteOrder binary.ByteOrder

	// 消息字节数
	// 默认为5000字节
	bufferBytes int

	// 是否携带心跳时间
	// 默认为false
	heartbeatTime bool
}

type Option func(o *options)

func defaultOptions() *options {
	opts := &options{
		byteOrder:     binary.BigEndian,
		bufferBytes:   etc.Get(defaultBufferBytesKey, defaultBufferBytes).Int(),
		heartbeatTime: etc.Get(defaultHeartbeatTimeKey, defaultHeartbeatTime).Bool(),
	}

	endian := etc.Get(defaultEndianKey, bigEndian).String()
	switch strings.ToLower(endian) {
	case littleEndian:
		opts.byteOrder = binary.LittleEndian
	case bigEndian:
		opts.byteOrder = binary.BigEndian
	}

	return opts
}

// WithByteOrder 设置字节序
func WithByteOrder(byteOrder binary.ByteOrder) Option {
	return func(o *options) { o.byteOrder = byteOrder }
}

// WithBufferBytes 设置消息字节数
func WithBufferBytes(bufferBytes int) Option {
	return func(o *options) { o.bufferBytes = bufferBytes }
}

// WithHeartbeatTime 是否携带心跳时间
func WithHeartbeatTime(heartbeatTime bool) Option {
	return func(o *options) { o.heartbeatTime = heartbeatTime }
}
