package client

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xbaseio/xbase/core/buffer"
	"github.com/xbaseio/xbase/internal/transporter/internal/def"
	"github.com/xbaseio/xbase/internal/transporter/internal/protocol"
	"github.com/xbaseio/xbase/mode"
	"github.com/xbaseio/xbase/utils/xtime"
	"github.com/xbaseio/xbase/xerrors"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

const (
	maxRetryTimes = 3
	dialTimeout   = 500 * time.Millisecond
	writeTimeout  = 3 * time.Second
)

type conn struct {
	cli *Client

	// 保护 conn / ctx / cancel / success / failure
	rw sync.RWMutex

	// 保护 close/drain 与 send 入队，避免 message 入队后无人释放
	sendMu sync.RWMutex

	conn  net.Conn
	state atomic.Int32

	queue   chan *message
	pending *pending

	failure chan struct{}
	success chan struct{}

	ctx    context.Context
	cancel context.CancelFunc

	lastFaultTime     atomic.Int64
	lastHeartbeatTime atomic.Int64
}

func newConn(cli *Client) *conn {
	c := &conn{}
	c.cli = cli
	c.state.Store(def.ConnClosed)
	c.queue = make(chan *message, 4096)
	c.pending = newPending()
	c.failure = make(chan struct{})
	c.success = make(chan struct{})
	c.lastFaultTime.Store(xtime.Now().Unix())

	return c
}

// 拨号
func (c *conn) dial() error {
	c.rw.Lock()
	defer c.rw.Unlock()

	if c.state.Load() == def.ConnOpened {
		return nil
	}

	if err := c.doDialLocked(); err != nil {
		c.markClosedLocked()
		c.signalFailureLocked()
		return err
	}

	c.signalSuccessLocked()
	return nil
}

// 执行拨号，调用方必须持有 c.rw.Lock()
func (c *conn) doDialLocked() error {
	var (
		retry int
		delay time.Duration
	)

	for {
		conn, err := net.DialTimeout("tcp", c.cli.opts.Addr, dialTimeout)
		if err != nil {
			retry++
			if retry >= maxRetryTimes {
				return err
			}

			if delay == 0 {
				delay = 5 * time.Millisecond
			} else {
				delay *= 2
			}

			if delay > time.Second {
				delay = time.Second
			}

			time.Sleep(delay)
			continue
		}

		if err = c.processLocked(conn); err != nil {
			_ = conn.Close()
			return err
		}

		return nil
	}
}

// 处理连接，调用方必须持有 c.rw.Lock()
func (c *conn) processLocked(conn net.Conn) error {
	ctx, cancel := context.WithCancel(context.Background())

	c.conn = conn
	c.ctx = ctx
	c.cancel = cancel
	c.state.Store(def.ConnOpened)
	c.lastHeartbeatTime.Store(xtime.Now().Unix())

	if err := c.handshake(ctx, conn); err != nil {
		cancel()
		_ = conn.Close()

		if c.conn == conn {
			c.conn = nil
		}

		c.state.Store(def.ConnClosed)
		return err
	}

	go c.read(ctx, conn)
	go c.write(ctx, conn)

	return nil
}

// 握手
func (c *conn) handshake(ctx context.Context, conn net.Conn) error {
	const seq = uint64(1)

	buf := protocol.EncodeHandshakeReq(seq, c.cli.opts.InsKind, c.cli.opts.InsID)
	if buf == nil {
		return xerrors.ErrInvalidMessage
	}
	defer buf.Release()

	if err := writeAllWithDeadline(conn, buf.Bytes(), writeTimeout); err != nil {
		return err
	}

	// 握手阶段直接读取握手响应，不依赖 pending，避免无缓冲 call 卡住 read goroutine。
	_ = conn.SetDeadline(time.Now().Add(defaultTimeout))
	defer func() {
		_ = conn.SetDeadline(time.Time{})
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
		}

		resp, err := protocol.ReaderBuffer(conn)
		if err != nil {
			return err
		}

		isHeartbeat, _, respSeq := protocol.ParseBuffer(resp.Bytes())
		resp.Release()

		if isHeartbeat {
			continue
		}

		if respSeq == seq {
			return nil
		}
	}
}

// 发送消息
func (c *conn) send(msg *message) error {
	if msg == nil {
		return nil
	}

	for {
		switch c.state.Load() {
		case def.ConnClosed:
			if mode.IsReleaseMode() && xtime.Now().Unix()-c.lastFaultTime.Load() < c.cli.opts.FaultInterval {
				return xerrors.ErrConnectionClosed
			}

			if err := c.dial(); err != nil {
				return err
			}

		case def.ConnHanged:
			if err := c.wait(); err != nil {
				return err
			}

		case def.ConnOpened:
			return c.enqueue(msg)

		default:
			return xerrors.ErrConnectionClosed
		}
	}
}

func (c *conn) enqueue(msg *message) error {
	c.sendMu.RLock()

	if c.state.Load() != def.ConnOpened {
		c.sendMu.RUnlock()
		return xerrors.ErrConnectionClosed
	}

	ctx := c.getContext()
	if ctx == nil {
		c.sendMu.RUnlock()
		return xerrors.ErrConnectionClosed
	}

	// 快路径：队列没满，不创建 timer。
	select {
	case c.queue <- msg:
		c.sendMu.RUnlock()
		return nil

	case <-ctx.Done():
		c.sendMu.RUnlock()
		return xerrors.ErrConnectionClosed

	default:
	}

	// 慢路径：队列满了才等待 writeTimeout。
	timer := time.NewTimer(writeTimeout)

	var retErr error
	var needClose bool

	select {
	case c.queue <- msg:
		retErr = nil

	case <-ctx.Done():
		retErr = xerrors.ErrConnectionClosed

	case <-timer.C:
		retErr = xerrors.ErrConnectionClosed
		needClose = true
	}

	stopTimer(timer)
	c.sendMu.RUnlock()

	if needClose {
		xlog.Logger().Warn("transporter client write queue timeout")
		c.close()
	}

	return retErr
}

// 读取数据
func (c *conn) read(ctx context.Context, conn net.Conn) {
	for {
		select {
		case <-ctx.Done():
			return

		default:
		}

		buf, err := protocol.ReaderBuffer(conn)
		if err != nil {
			c.retry(conn)
			return
		}

		c.lastHeartbeatTime.Store(xtime.Now().Unix())

		isHeartbeat, _, seq := protocol.ParseBuffer(buf.Bytes())
		if isHeartbeat {
			buf.Release()
			continue
		}

		call, ok := c.pending.extract(seq)
		if !ok {
			buf.Release()
			continue
		}

		select {
		case call <- buf:
		case <-ctx.Done():
			buf.Release()
		}
	}
}

// 写入数据
func (c *conn) write(ctx context.Context, conn net.Conn) {
	ticker := time.NewTicker(def.HeartbeatInterval)

	defer func() {
		ticker.Stop()

		// write 协程退出时兜底清理队列，避免 message 泄漏。
		c.sendMu.Lock()
		c.drainQueue()
		c.sendMu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return

		case t := <-ticker.C:
			deadline := t.Add(-2 * def.HeartbeatInterval).Unix()
			if c.lastHeartbeatTime.Load() < deadline {
				xlog.Logger().Warn("connection heartbeat timeout")
				c.retry(conn)
				return
			}

			if err := writeAllWithDeadline(conn, protocol.Heartbeat(), writeTimeout); err != nil {
				xlog.Logger().Warn("write heartbeat message error", zap.Error(err))
				c.retry(conn)
				return
			}

		case msg := <-c.queue:
			if msg == nil {
				continue
			}

			if ok := c.doWrite(conn, msg); !ok {
				return
			}
		}
	}
}

// 执行写入数据
func (c *conn) doWrite(conn net.Conn, msg *message) bool {
	if msg.seq != 0 {
		if !msg.state.CompareAndSwap(statePending, stateSent) {
			// 说明这个消息已经被取消，不应该导致 write goroutine 退出。
			c.cli.release(msg)
			return true
		}

		c.pending.store(msg.seq, msg.call)
	}

	ok := msg.buf.Visit(func(node *buffer.NocopyNode) bool {
		if node == nil {
			return true
		}

		data := node.Bytes()
		if len(data) == 0 {
			return true
		}

		if err := writeAllWithDeadline(conn, data, writeTimeout); err != nil {
			xlog.Logger().Warn("write transporter message error", zap.Error(err))
			return false
		}

		return true
	})

	if !ok && msg.seq != 0 {
		c.pending.delete(msg.seq)
	}

	c.cli.release(msg)

	if !ok {
		c.retry(conn)
		return false
	}

	return true
}

// 重试拨号
func (c *conn) retry(conn net.Conn) {
	if !c.state.CompareAndSwap(def.ConnOpened, def.ConnHanged) {
		return
	}

	c.rw.RLock()
	curConn := c.conn
	cancel := c.cancel
	c.rw.RUnlock()

	if curConn == conn {
		_ = conn.Close()
	}

	if cancel != nil {
		cancel()
	}

	if err := c.dial(); err != nil {
		xlog.Logger().Warn("retry dial failed", zap.Error(err))
	}
}

// 关闭连接
func (c *conn) close() {
	if c.state.Swap(def.ConnClosed) == def.ConnClosed {
		return
	}

	c.lastFaultTime.Store(xtime.Now().Unix())

	c.rw.Lock()
	conn := c.conn
	cancel := c.cancel
	c.conn = nil
	c.cancel = nil
	c.rw.Unlock()

	if cancel != nil {
		cancel()
	}

	if conn != nil {
		_ = conn.Close()
	}

	c.sendMu.Lock()
	c.drainQueue()
	c.sendMu.Unlock()

	c.signalFailure()
}

// 等待重连
func (c *conn) wait() error {
	c.rw.RLock()
	state := c.state.Load()
	failure := c.failure
	success := c.success
	c.rw.RUnlock()

	switch state {
	case def.ConnOpened:
		return nil

	case def.ConnHanged:
		select {
		case <-failure:
			return xerrors.ErrConnectionClosed

		case <-success:
			return nil
		}
	}

	return xerrors.ErrConnectionClosed
}

// 删除发送消息
func (c *conn) delete(msg *message) {
	if msg == nil {
		return
	}

	if !msg.state.CompareAndSwap(statePending, stateCanceled) {
		c.pending.delete(msg.seq)
	}
}

func (c *conn) markClosedLocked() {
	if c.state.Swap(def.ConnClosed) != def.ConnClosed {
		c.lastFaultTime.Store(xtime.Now().Unix())
	}

	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *conn) signalSuccessLocked() {
	close(c.success)
	c.success = make(chan struct{})
}

func (c *conn) signalFailureLocked() {
	close(c.failure)
	c.failure = make(chan struct{})
}

func (c *conn) signalFailure() {
	c.rw.Lock()
	c.signalFailureLocked()
	c.rw.Unlock()
}

func (c *conn) getContext() context.Context {
	c.rw.RLock()
	ctx := c.ctx
	c.rw.RUnlock()
	return ctx
}

func (c *conn) drainQueue() {
	for {
		select {
		case msg := <-c.queue:
			if msg != nil {
				c.cli.release(msg)
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
