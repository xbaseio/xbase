package xnet

import (
	"io"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/utils/xnetpoll"
)

// processIO 处理连接上的 IO 事件。
func (c *conn) processIO(_ int, ev xnetpoll.IOEvent, _ xnetpoll.IOFlags) error {
	el := c.loop

	// 先检查是否出现了非预期的非 IO 事件。
	// 对于这类事件，直接关闭连接即可。
	if ev&(xnetpoll.ErrEvents|unix.EPOLLRDHUP) != 0 && ev&xnetpoll.ReadWriteEvents == 0 {
		// 连接已经损坏，不再尝试写出剩余数据。
		c.outboundBuffer.Release()
		return el.close(c, io.EOF)
	}

	// 其次，优先处理 EPOLLOUT，再处理 EPOLLIN。
	// 无论当前连接是否仍然存活，写事件的优先级都高于读事件：
	//
	// 1. 当连接仍然存活且系统负载较高时，
	//    应优先把待发送数据尽快写回对端，再继续读取和处理新的请求。
	// 2. 当连接已经失效时，
	//    也应先尝试把待发送数据写回，再关闭连接。
	//
	// 因此这里在收到 EPOLLOUT 时先执行 eventloop.write，
	// 因为它可以同时兼顾上述两种情况。
	if ev&(xnetpoll.WriteEvents|xnetpoll.ErrEvents) != 0 {
		if err := el.write(c); err != nil {
			return err
		}
	}

	// 在检查 EPOLLRDHUP 之前先处理 EPOLLIN，
	// 因为 socket 缓冲区里可能仍然有待读取的数据。
	if ev&(xnetpoll.ReadEvents|xnetpoll.ErrEvents) != 0 {
		if err := el.read(c); err != nil {
			return err
		}
	}

	// 最后检查 EPOLLRDHUP。
	// 该事件表示对端已经关闭连接，或者关闭了连接的写半边。
	if ev&unix.EPOLLRDHUP != 0 && c.opened {
		// 没有可读数据的 EPOLLRDHUP，直接关闭连接。
		if ev&unix.EPOLLIN == 0 {
			return el.close(c, io.EOF)
		}

		// 收到 EPOLLIN|EPOLLRDHUP，说明对端关闭前可能还有数据残留在 socket buffer 中。
		// 如果前一次 eventloop.read 没有把缓冲区读空，这里再补读一次，确保数据被完全取走。
		c.isEOF = true
		return el.read(c)
	}

	return nil
}
