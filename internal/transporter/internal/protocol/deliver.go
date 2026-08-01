package protocol

import (
	"encoding/binary"
	"encoding/json"
	"io"

	"github.com/xbaseio/xbase/core/buffer"
	"github.com/xbaseio/xbase/internal/transporter/internal/route"
	"github.com/xbaseio/xbase/xerrors"
)

const (
	deliverReqBytes = defaultSizeBytes + defaultHeaderBytes + defaultRouteBytes + defaultSeqBytes + b64 + b64 + b32
	deliverResBytes = defaultSizeBytes + defaultHeaderBytes + defaultRouteBytes + defaultSeqBytes + defaultCodeBytes
)

// EncodeDeliverReq 编码投递消息请求
// 协议：size + header + route + seq + cid + uid + metadataSize + metadata + <message packet>
func EncodeDeliverReq(seq uint64, cid int64, uid int64, metadata map[string]string, buf buffer.Buffer) *buffer.NocopyBuffer {
	var metadataData []byte
	if len(metadata) > 0 {
		metadataData, _ = json.Marshal(metadata)
	}
	writer := buffer.MallocWriter(deliverReqBytes)
	writer.WriteUint32s(binary.BigEndian, uint32(deliverReqBytes-defaultSizeBytes+len(metadataData)+buf.Len()))
	writer.WriteUint8s(dataBit)
	writer.WriteUint8s(route.Deliver)
	writer.WriteUint64s(binary.BigEndian, seq)
	writer.WriteInt64s(binary.BigEndian, cid, uid)
	writer.WriteUint32s(binary.BigEndian, uint32(len(metadataData)))

	return buffer.NewNocopyBuffer(writer, metadataData, buf)
}

// DecodeDeliverReq 解码投递消息请求
func DecodeDeliverReq(data []byte) (seq uint64, cid int64, uid int64, metadata map[string]string, message []byte, err error) {
	reader := buffer.NewReader(data)

	if _, err = reader.Seek(defaultSizeBytes+defaultHeaderBytes+defaultRouteBytes, io.SeekStart); err != nil {
		return
	}

	if seq, err = reader.ReadUint64(binary.BigEndian); err != nil {
		return
	}

	if cid, err = reader.ReadInt64(binary.BigEndian); err != nil {
		return
	}

	if uid, err = reader.ReadInt64(binary.BigEndian); err != nil {
		return
	}

	metadataSize, readErr := reader.ReadUint32(binary.BigEndian)
	if readErr != nil {
		err = readErr
		return
	}
	messageOffset := deliverReqBytes + int(metadataSize)
	if messageOffset < deliverReqBytes || messageOffset > len(data) {
		err = xerrors.ErrInvalidMessage
		return
	}
	if metadataSize > 0 {
		if err = json.Unmarshal(data[deliverReqBytes:messageOffset], &metadata); err != nil {
			return
		}
	}

	message = data[messageOffset:]

	return
}

// EncodeDeliverRes 编码投递消息响应
// 协议：size + header + route + seq + code
func EncodeDeliverRes(seq uint64, code uint16) *buffer.NocopyBuffer {
	writer := buffer.MallocWriter(deliverResBytes)
	writer.WriteUint32s(binary.BigEndian, uint32(deliverResBytes-defaultSizeBytes))
	writer.WriteUint8s(dataBit)
	writer.WriteUint8s(route.Deliver)
	writer.WriteUint64s(binary.BigEndian, seq)
	writer.WriteUint16s(binary.BigEndian, code)

	return buffer.NewNocopyBuffer(writer)
}

// DecodeDeliverRes 解码投递消息响应
// 协议：size + header + route + seq + code
func DecodeDeliverRes(data []byte) (code uint16, err error) {
	if len(data) != deliverResBytes {
		err = xerrors.ErrInvalidMessage
		return
	}

	reader := buffer.NewReader(data)

	if _, err = reader.Seek(-defaultCodeBytes, io.SeekEnd); err != nil {
		return
	}

	if code, err = reader.ReadUint16(binary.BigEndian); err != nil {
		return
	}

	return
}
