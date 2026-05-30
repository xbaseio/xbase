package packet

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/xbaseio/xbase/core/buffer"
	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/xerrors"
)

const (
	dataBit = 0 << 7 // 数据标识
)

type NocopyReader interface {
	// Next returns a slice containing the next n bytes from the buffer,
	// advancing the buffer as if the bytes had been returned by Read.
	Next(n int) (p []byte, err error)

	// Peek returns the next n bytes without advancing the reader.
	Peek(n int) (buf []byte, err error)

	// Release the memory space occupied by all read slices.
	Release() (err error)

	Slice(n int) (r NocopyReader, err error)
}

type Packer interface {
	// ReadBuffer 以buffer的形式读取消息
	ReadBuffer(reader io.Reader) (buffer.Buffer, error)
	// PackBuffer 以buffer的形式打包消息
	PackBuffer(message *Message) (*buffer.NocopyBuffer, error)
	// ReadMessage 读取消息
	ReadMessage(reader io.Reader) ([]byte, error)
	// PackMessage 打包消息
	PackMessage(message *Message) ([]byte, error)
	// UnpackMessage 解包消息
	UnpackMessage(data []byte) (*Message, int, error)
}

type defaultPacker struct {
	opts *options
}

func NewPacker(opts ...Option) *defaultPacker {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}

	if o.bufferBytes < 0 {
		log.Fatalf("the number of buffer bytes must be greater than or equal to 0, and give %d", o.bufferBytes)
	}

	return &defaultPacker{
		opts: o,
	}
}

// ReadBuffer 以buffer的形式读取消息
func (p *defaultPacker) ReadBuffer(reader io.Reader) (buffer.Buffer, error) {
	buf1 := buffer.MallocBytes(defaultSizeBytes)
	defer buf1.Release()

	if _, err := io.ReadFull(reader, buf1.Bytes()); err != nil {
		return nil, err
	}

	size := p.opts.byteOrder.Uint32(buf1.Bytes())

	if size == 0 {
		return nil, nil
	}

	buf2 := buffer.MallocBytes(int(defaultSizeBytes + size))
	data := buf2.Bytes()

	copy(data[:defaultSizeBytes], buf1.Bytes())

	if _, err := io.ReadFull(reader, data[defaultSizeBytes:]); err != nil {
		buf2.Release()
		return nil, err
	}

	return buf2, nil
}

// PackBuffer 以buffer的形式打包消息
func (p *defaultPacker) PackBuffer(message *Message) (*buffer.NocopyBuffer, error) {
	if len(message.Buffer) > p.opts.bufferBytes {
		return nil, xerrors.ErrMessageTooLarge
	}
	// -------------------------
	// 2. 固定 header = 4字段 * 4bytes
	// -------------------------

	totalLen := defaultHeaderSize + len(message.Buffer)

	if totalLen > p.opts.bufferBytes || totalLen > math.MaxInt32 {
		return nil, xerrors.ErrMessageTooLarge
	}
	writer := buffer.MallocWriter(defaultHeaderSize)
	writer.WriteInt32s(p.opts.byteOrder, int32(totalLen))
	writer.WriteInt32s(p.opts.byteOrder, dataBit)

	writer.WriteInt32s(p.opts.byteOrder, message.GameID)
	writer.WriteInt32s(p.opts.byteOrder, message.MessageID)
	writer.WriteInt32s(p.opts.byteOrder, message.Seq)

	return buffer.NewNocopyBuffer(writer, message.Buffer), nil
}

// ReadMessage 读取消息
func (p *defaultPacker) ReadMessage(reader io.Reader) ([]byte, error) {
	buf := make([]byte, defaultSizeBytes)

	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, err
	}

	size := p.opts.byteOrder.Uint32(buf)

	if size == 0 {
		return nil, nil
	}

	data := make([]byte, int(defaultSizeBytes+size))

	copy(data[:defaultSizeBytes], buf)

	if _, err := io.ReadFull(reader, data[defaultSizeBytes:]); err != nil {
		return nil, err
	}

	return data, nil
}

// 无拷贝读取消息
func (p *defaultPacker) nocopyReadMessage(reader NocopyReader) ([]byte, error) {
	buf, err := reader.Peek(defaultSizeBytes)
	if err != nil {
		return nil, err
	}

	var size uint32

	if p.opts.byteOrder == binary.BigEndian {
		size = binary.BigEndian.Uint32(buf)
	} else {
		size = binary.LittleEndian.Uint32(buf)
	}

	if size == 0 {
		return nil, nil
	}

	n := int(defaultSizeBytes + size)

	r, err := reader.Slice(n)
	if err != nil {
		return nil, err
	}

	buf, err = r.Next(n)
	if err != nil {
		return nil, err
	}

	if err = reader.Release(); err != nil {
		return nil, err
	}

	return buf, nil
}

// PackMessage 打包消息
func (p *defaultPacker) PackMessage(message *Message) ([]byte, error) {

	// -------------------------
	// 1. body 校验
	// -------------------------
	if len(message.Buffer) > p.opts.bufferBytes {
		return nil, xerrors.ErrMessageTooLarge
	}

	// -------------------------
	// 2. 固定 header = 4字段 * 4bytes
	// -------------------------

	totalLen := defaultHeaderSize + len(message.Buffer)

	// 防止 int32 溢出
	if totalLen > math.MaxInt32 {
		return nil, xerrors.ErrMessageTooLarge
	}

	// -------------------------
	// 3. 分配 buffer
	// -------------------------
	buf := make([]byte, 4+totalLen) // +4 是 length

	offset := 0

	// -------------------------
	// 4. 写 totalLen
	// -------------------------
	p.opts.byteOrder.PutUint32(buf[offset:], uint32(totalLen))
	offset += 4

	// -------------------------
	// 5. dataBit
	// -------------------------
	p.opts.byteOrder.PutUint32(buf[offset:], uint32(dataBit))
	offset += 4

	// -------------------------
	// 6. GameID
	// -------------------------
	p.opts.byteOrder.PutUint32(buf[offset:], uint32(message.GameID))
	offset += 4

	// -------------------------
	// 7. messageID
	// -------------------------
	p.opts.byteOrder.PutUint32(buf[offset:], uint32(message.MessageID))
	offset += 4

	// -------------------------
	// -------------------------
	// 8. seq
	// -------------------------
	p.opts.byteOrder.PutUint32(buf[offset:], uint32(message.Seq))
	offset += 4
	// -------------------------
	// 9. body
	// -------------------------
	copy(buf[offset:], message.Buffer)

	return buf, nil
}

func (p *defaultPacker) UnpackMessage(data []byte) (*Message, int, error) {

	// -------------------------
	// 1. 至少要有 size 字段
	// -------------------------
	if len(data) < defaultSizeBytes {
		return nil, 0, nil // half packet
	}

	offset := 0

	// -------------------------
	// 2. size（PackBuffer 内层长度；ReadMessage 首 4 字节与之相同）
	// -------------------------
	size := int(p.opts.byteOrder.Uint32(data[offset:]))
	offset += 4

	if size < defaultHeaderSize || size > p.opts.bufferBytes {
		return nil, 0, xerrors.ErrInvalidMessage
	}

	if len(data) < size {
		return nil, 0, nil // half packet
	}

	// -------------------------
	// 3. dataBit
	// -------------------------
	dataBitVal := int32(p.opts.byteOrder.Uint32(data[offset:]))
	offset += 4

	if dataBitVal&int32(dataBit) != int32(dataBit) {
		return nil, 0, xerrors.ErrInvalidMessage
	}

	// -------------------------
	// 4. GameID
	// -------------------------
	gameID := int32(p.opts.byteOrder.Uint32(data[offset:]))
	offset += 4

	// -------------------------
	// 5. messageID
	// -------------------------
	messageID := int32(p.opts.byteOrder.Uint32(data[offset:]))
	offset += 4

	// -------------------------
	// 6. seq
	// -------------------------
	seq := int32(p.opts.byteOrder.Uint32(data[offset:]))
	offset += 4

	// -------------------------
	// 7. body（zero-copy）
	// -------------------------
	body := data[offset:size]

	// -------------------------
	// 8. 组装 message
	// -------------------------
	msg := &Message{
		GameID:    gameID,
		Seq:       seq,
		MessageID: messageID,
		Buffer:    body,
	}

	return msg, size, nil
}
