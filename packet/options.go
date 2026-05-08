package packet

import (
	"encoding/binary"
	"strings"

	"github.com/xbaseio/xbase/etc"
)

const (
	littleEndian = "little"
	bigEndian    = "big"
)

const (
	defaultSizeBytes   = 4
	defaultBufferBytes = 65535
	defaultHeaderSize  = 16
)

const (
	defaultEndianKey      = "etc.packet.byteOrder"
	defaultBufferBytesKey = "etc.packet.bufferBytes"
)

type options struct {
	// 字节序
	// 默认为binary.LittleEndian
	byteOrder binary.ByteOrder

	// 消息字节数
	// 默认为5000字节
	bufferBytes int
}

type Option func(o *options)

func defaultOptions() *options {
	opts := &options{
		byteOrder:   binary.BigEndian,
		bufferBytes: etc.Get(defaultBufferBytesKey, defaultBufferBytes).Int(),
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
