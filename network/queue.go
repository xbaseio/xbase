package network

import "time"

const (
	// DefaultRecvQueueSize 收包队列默认容量；满则关闭连接，避免 read 协程阻塞。
	DefaultRecvQueueSize = 1024
	// DefaultWriteQueueSize 写队列默认容量。
	DefaultWriteQueueSize = 4096
	// DefaultWriteEnqueueTimeout 写队列入队超时；超时后关闭连接，与 transporter 对齐。
	DefaultWriteEnqueueTimeout = 3 * time.Second
)

// TryEnqueueRecv 非阻塞写入收包队列；队列满返回 false。
func TryEnqueueRecv(recvQ chan []byte, data []byte) bool {
	frame := append([]byte(nil), data...)

	select {
	case recvQ <- frame:
		return true
	default:
		return false
	}
}
