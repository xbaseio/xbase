package server

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/cluster"
	"github.com/xbaseio/xbase/core/buffer"
	"github.com/xbaseio/xbase/internal/transporter/internal/def"
	"github.com/xbaseio/xbase/internal/transporter/internal/protocol"
	"github.com/xbaseio/xbase/utils/xtime"
	"github.com/xbaseio/xbase/xerrors"
	"github.com/xbaseio/xbase/xlog"
)

const writeTimeout = 3 * time.Second

type Conn struct {
	ctx    context.Context
	cancel context.CancelFunc
	server *Server
	conn   net.Conn
	state  int32

	chWrite     chan *buffer.NocopyBuffer
	chHeartbeat chan []byte

	// 这个锁主要不是为了保护 channel。
	// 它是为了保证 close drain 队列时，不会和 Send 并发导致 NocopyBuffer 泄漏。
	sendMu sync.RWMutex

	lastHeartbeatTime int64

	InsKind cluster.Kind
	InsID   string
}

func newConn(server *Server, conn net.Conn) *Conn {
	c := &Conn{}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	c.conn = conn
	c.server = server
	c.state = def.ConnOpened

	c.chWrite = make(chan *buffer.NocopyBuffer, 4096)

	// 心跳队列单独分离，优先写，不允许无限堆积。
	c.chHeartbeat = make(chan []byte, 16)

	c.lastHeartbeatTime = xtime.Now().Unix()

	return c
}

// start 启动读写协程
func (c *Conn) start() {
	if atomic.LoadInt32(&c.state) == def.ConnClosed {
		return
	}

	go c.read()
	go c.write()
}

// Send 发送业务消息
//
// 约定：
// Send 成功后，buf 由 Conn 接管，write 协程负责 Release。
// Send 失败时，Send 内部负责 Release。
// 调用方不要再手动 Release。
func (c *Conn) Send(buf *buffer.NocopyBuffer) error {
	if buf == nil {
		return nil
	}

	c.sendMu.RLock()

	if atomic.LoadInt32(&c.state) == def.ConnClosed {
		c.sendMu.RUnlock()
		buf.Release()
		return xerrors.ErrConnectionClosed
	}

	// 快路径：队列没满时不创建 timer，减少高频发包的分配开销。
	select {
	case c.chWrite <- buf:
		c.sendMu.RUnlock()
		return nil

	case <-c.ctx.Done():
		c.sendMu.RUnlock()
		buf.Release()
		return xerrors.ErrConnectionClosed

	default:
	}

	// 慢路径：队列满了才等 writeTimeout。
	timer := time.NewTimer(writeTimeout)

	var sent bool
	var needClose bool
	var retErr error

	select {
	case <-c.ctx.Done():
		retErr = xerrors.ErrConnectionClosed

	case c.chWrite <- buf:
		sent = true
		retErr = nil

	case <-timer.C:
		needClose = true
		retErr = xerrors.ErrConnectionClosed
	}

	stopTimer(timer)

	c.sendMu.RUnlock()

	if !sent {
		buf.Release()
	}

	if needClose {
		xlog.Logger().Warn("connection business write channel timeout")
		_ = c.close(true)
	}

	return retErr
}

// sendHeartbeat 发送心跳响应
func (c *Conn) sendHeartbeat() error {
	c.sendMu.RLock()

	if atomic.LoadInt32(&c.state) == def.ConnClosed {
		c.sendMu.RUnlock()
		return xerrors.ErrConnectionClosed
	}

	var needClose bool
	var retErr error

	select {
	case <-c.ctx.Done():
		retErr = xerrors.ErrConnectionClosed

	case c.chHeartbeat <- protocol.Heartbeat():
		retErr = nil

	default:
		// 心跳队列满，说明写端已经堵了。
		// 不阻塞 read goroutine，直接关连接。
		needClose = true
		retErr = xerrors.ErrConnectionClosed
	}

	c.sendMu.RUnlock()

	if needClose {
		xlog.Logger().Warn("connection heartbeat channel full")
		_ = c.close(true)
	}

	return retErr
}

// close 关闭连接
func (c *Conn) close(isNeedRecycle ...bool) error {
	if !atomic.CompareAndSwapInt32(&c.state, def.ConnOpened, def.ConnClosed) {
		return xerrors.ErrConnectionClosed
	}

	c.cancel()

	conn := c.conn
	err := conn.Close()

	// 等待正在 Send/sendHeartbeat 的 goroutine 退出。
	// 然后清理业务队列里还没写出去的 buffer，避免 NocopyBuffer 泄漏。
	c.sendMu.Lock()
	c.drainWriteChan()
	c.sendMu.Unlock()

	if len(isNeedRecycle) > 0 && isNeedRecycle[0] {
		c.server.recycle(conn)
	}

	return err
}

// read 读取消息
func (c *Conn) read() {
	conn := c.conn

	for {
		select {
		case <-c.ctx.Done():
			return

		default:
		}

		isHeartbeat, routeID, _, data, err := protocol.ReadMessage(conn)
		if err != nil {
			_ = c.close(true)
			return
		}

		if atomic.LoadInt32(&c.state) == def.ConnClosed {
			return
		}

		atomic.StoreInt64(&c.lastHeartbeatTime, xtime.Now().Unix())

		// 心跳在 Conn 层单独处理，不进入业务 route。
		if isHeartbeat {
			if err := c.sendHeartbeat(); err != nil {
				return
			}
			continue
		}

		handler := c.server.getHandler(routeID)
		if handler == nil {
			continue
		}

		if err := handler(c, data); err != nil && !xerrors.Is(err, xerrors.ErrNotFoundUserLocation) {
			xlog.Sugar().Warnf("process route %d message failed: %v", routeID, err)
		}
	}
}

// write 写入消息
func (c *Conn) write() {
	conn := c.conn
	ticker := time.NewTicker(def.HeartbeatInterval)

	defer func() {
		ticker.Stop()

		// write 协程退出时，再兜底清理一次业务队列。
		// close() 里也会 drain，一条消息只会被一个 receiver 取到，不会重复 Release。
		c.drainWriteChan()
	}()

	for {
		// 优先处理心跳，避免业务包太多时心跳响应被压住。
		select {
		case hb := <-c.chHeartbeat:
			if err := writeAllWithDeadline(conn, hb, writeTimeout); err != nil {
				xlog.Sugar().Warnf("write heartbeat message error: %v", err)
				_ = c.close(true)
				return
			}
			continue

		default:
		}

		select {
		case <-c.ctx.Done():
			return

		case <-ticker.C:
			deadline := xtime.Now().Add(-2 * def.HeartbeatInterval).Unix()
			if atomic.LoadInt64(&c.lastHeartbeatTime) < deadline {
				xlog.Logger().Warn("connection heartbeat timeout")
				_ = c.close(true)
				return
			}

		case hb := <-c.chHeartbeat:
			if err := writeAllWithDeadline(conn, hb, writeTimeout); err != nil {
				xlog.Sugar().Warnf("write heartbeat message error: %v", err)
				_ = c.close(true)
				return
			}

		case buf := <-c.chWrite:
			if buf == nil {
				continue
			}

			ok := buf.Visit(func(node *buffer.NocopyNode) bool {
				if node == nil {
					return true
				}

				data := node.Bytes()
				if len(data) == 0 {
					return true
				}

				if err := writeAllWithDeadline(conn, data, writeTimeout); err != nil {
					xlog.Sugar().Warnf("write business message error: %v", err)
					return false
				}

				return true
			})

			buf.Release()

			if !ok {
				_ = c.close(true)
				return
			}
		}
	}
}

// drainWriteChan 清理业务发送队列里还没有写出去的 buffer
func (c *Conn) drainWriteChan() {
	for {
		select {
		case buf := <-c.chWrite:
			if buf != nil {
				buf.Release()
			}

		default:
			return
		}
	}
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}

	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

// writeAllWithDeadline 确保完整写入，并给底层 conn.Write 设置超时。
func writeAllWithDeadline(conn net.Conn, data []byte, timeout time.Duration) error {
	if conn == nil {
		return xerrors.ErrConnectionClosed
	}

	if len(data) == 0 {
		return nil
	}

	if timeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	}

	return writeAll(conn, data)
}

// writeAll 确保完整写入
func writeAll(conn net.Conn, data []byte) error {
	for len(data) > 0 {
		n, err := conn.Write(data)
		if err != nil {
			return err
		}

		if n <= 0 {
			return io.ErrShortWrite
		}

		data = data[n:]
	}

	return nil
}
