//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package xnet

import (
	"io"

	"golang.org/x/sys/unix"

	"github.com/xbaseio/xbase/utils/xnetpoll"
)

// processIO 处理 kqueue 触发的 IO 事件。
func (c *conn) processIO(_ int, filter xnetpoll.IOEvent, flags xnetpoll.IOFlags) (err error) {
	el := c.loop

	// ===== 基础读写事件处理 =====
	switch filter {
	case unix.EVFILT_READ:
		err = el.read(c)

	case unix.EVFILT_WRITE:
		err = el.write(c)
	}

	// ===== EOF 处理 =====
	// EV_EOF 表示对端已经关闭连接。
	// 注意：必须在 read/write 之后处理，确保缓冲区数据不会丢失。
	if flags&unix.EV_EOF != 0 && c.opened && err == nil {

		switch filter {

		case unix.EVFILT_READ:
			// 收到 READ|EOF，但上一次 read 没有把 socket buffer 读干净。
			// 这里强制再读一次，确保数据全部消费完。
			c.isEOF = true
			err = el.read(c)

		case unix.EVFILT_WRITE:
			// macOS 行为说明：
			// TCP：EOF 只会触发一次 READ|EOF
			// Unix Socket（ET 模式）：可能触发两次事件：
			//   1) WRITE|EOF
			//   2) READ|EOF
			// 这里 WRITE|EOF 再走一次 write，确保发送缓冲处理完成
			err = el.write(c)

		default:
			// 其他情况：直接关闭连接
			// 已经是断开连接，不需要再尝试写数据
			c.outboundBuffer.Release()
			err = el.close(c, io.EOF)
		}
	}

	return
}
