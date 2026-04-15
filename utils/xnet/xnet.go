// Package xnet 实现了一个高性能、轻量级、非阻塞、
// 事件驱动的网络框架，使用纯 Go 编写。

package xnet

import (
	"context"
	"io"
	"net"
	"net/url"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xbuffer/xring"
	"github.com/xbaseio/xbase/utils/xgfd"
	"github.com/xbaseio/xbase/utils/xmath"
	"github.com/xbaseio/xbase/xerrors"
)

// Action 是在事件完成后的操作。
type Action int

const (
	// None 表示事件后不应发生任何操作。
	None Action = iota

	// Close 关闭连接。
	Close

	// Shutdown 关闭引擎。
	Shutdown
)

// Engine 表示一个引擎上下文，提供一些函数。
type Engine struct {
	// eng 是内部引擎结构体。
	eng *engine
}

// Validate 检查引擎是否可用。
func (e Engine) Validate() error {
	if e.eng == nil || len(e.eng.listeners) == 0 {
		return xerrors.ErrEmptyEngine
	}
	if e.eng.isShutdown() {
		return xerrors.ErrEngineInShutdown
	}
	return nil
}

// CountConnections 计算当前活跃连接的数量并返回。
func (e Engine) CountConnections() (count int) {
	if e.Validate() != nil {
		return -1
	}

	e.eng.eventLoops.iterate(func(_ int, el *eventloop) bool {
		count += int(el.countConn())
		return true
	})
	return
}

// Register 根据 WithLoadBalancing 设置的算法选择事件循环，
// 将新连接注册到该事件循环中。
// 您应该调用 NewNetConnContext 或 NewNetAddrContext 中的一个，
// 并将返回的上下文传递给此方法。如果上下文中同时存在 net.Conn 和 net.Addr，
// 则 net.Conn 优先。
//
// 注意，如果您计划稍后从其他地方调用此方法，
// 需要在启动引擎时切换到默认 RoundRobin 以外的负载均衡算法，
// 以避免数据竞争问题。
func (e Engine) Register(ctx context.Context) (<-chan RegisteredResult, error) {
	if err := e.Validate(); err != nil {
		return nil, err
	}

	if e.eng.eventLoops.len() == 0 {
		return nil, xerrors.ErrEmptyEngine
	}

	c, ok := FromNetConnContext(ctx)
	if ok {
		return e.eng.eventLoops.next(c.RemoteAddr()).Enroll(ctx, c)
	}

	addr, ok := FromNetAddrContext(ctx)
	if ok {
		return e.eng.eventLoops.next(addr).Register(ctx, addr)
	}

	return nil, xerrors.ErrInvalidNetworkAddress
}

// Dup 返回监听器的底层文件描述符的副本。
// 调用者有责任在完成后关闭 dupFD。
// 关闭监听器不会影响 dupFD，关闭 dupFD 也不会影响监听器。
//
// 注意，此方法仅在引擎只有一个监听器时可用。
func (e Engine) Dup() (fd int, err error) {
	if err := e.Validate(); err != nil {
		return -1, err
	}

	if len(e.eng.listeners) > 1 {
		return -1, xerrors.ErrUnsupportedOp
	}

	for _, ln := range e.eng.listeners {
		fd, err = ln.dup()
	}

	return
}

// DupListener 类似于 Dup，但它复制具有给定网络和地址的监听器。
// 这在有多个监听器时很有用。
func (e Engine) DupListener(network, addr string) (int, error) {
	if err := e.Validate(); err != nil {
		return -1, err
	}

	for _, ln := range e.eng.listeners {
		if ln.network == network && ln.address == addr {
			return ln.dup()
		}
	}

	return -1, xerrors.ErrInvalidNetworkAddress
}

// Stop 优雅地关闭此引擎而不中断任何活跃的事件循环，
// 它无限期等待连接和事件循环关闭，然后关闭。
func (e Engine) Stop(ctx context.Context) error {
	if err := e.Validate(); err != nil {
		return err
	}

	e.eng.shutdown(nil)

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if e.eng.isShutdown() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

/*
type asyncCmdType uint8

const (
	asyncCmdClose = iota + 1
	asyncCmdWake
	asyncCmdWrite
	asyncCmdWritev
)

type asyncCmd struct {
	fd  xgfd.GFD
	typ asyncCmdType
	cb  AsyncCallback
	param any
}

// AsyncWrite 异步地将数据写入给定的连接。
func (e Engine) AsyncWrite(fd xgfd.GFD, p []byte, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdWrite, cb: cb, param: p}, false)
}

// AsyncWritev 类似于 AsyncWrite，但它接受字节切片的切片。
func (e Engine) AsyncWritev(fd xgfd.GFD, batch [][]byte, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdWritev, cb: cb, param: batch}, false)
}

// Close 关闭给定的连接。
func (e Engine) Close(fd xgfd.GFD, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdClose, cb: cb}, false)
}

// Wake 唤醒给定的连接。
func (e Engine) Wake(fd xgfd.GFD, cb AsyncCallback) error {
	if err := e.Validate(); err != nil {
		return err
	}

	return e.eng.sendCmd(&asyncCmd{fd: fd, typ: asyncCmdWake, cb: cb}, true)
}
*/

// Reader 是一个接口，由 Conn 必须实现的多个读取方法组成。
//
// 注意，此接口中的方法对于并发使用不是并发安全的，
// 您必须在 EventHandler 的任何方法中调用它们。
type Reader interface {
	io.Reader
	io.WriterTo

	// Next 返回接下来的 n 个字节并推进入站缓冲区。
	// buf 不得在新 goroutine 中使用。否则，请使用 Read。
	//
	// 如果可用字节数少于请求的数量，
	// 返回 (0, io.ErrShortBuffer)。
	Next(n int) (buf []byte, err error)

	// Peek 返回接下来的 n 个字节而不推进入站缓冲区，
	// 返回的字节在调用 Discard 之前保持有效。
	// buf 不得在新 goroutine 中使用，也不得在调用 Discard 后使用，
	// 请手动复制 buf 或使用 Read。
	//
	// 如果可用字节数少于请求的数量，
	// 返回 (0, io.ErrShortBuffer)。
	Peek(n int) (buf []byte, err error)

	// Discard 使用接下来的 n 个字节推进入站缓冲区，返回丢弃的字节数。
	Discard(n int) (discarded int, err error)

	// InboundBuffered 返回可以从当前缓冲区读取的字节数。
	InboundBuffered() int
}

// Writer 是一个接口，由 Conn 必须实现的多个写入方法组成。
type Writer interface {
	io.Writer     // 不是并发安全的
	io.ReaderFrom // 不是并发安全的

	// SendTo 将消息传输到给定的地址，不是并发安全的。
	// 仅适用于 UDP 套接字，在非 UDP 套接字上调用时将返回 ErrUnsupportedOp。
	// 此方法应仅在需要通过 UDP 套接字向特定地址发送消息时使用，
	// 否则应使用 Conn.Write()。
	SendTo(buf []byte, addr net.Addr) (n int, err error)

	// Writev 同步地将多个字节切片写入远程，不是并发安全的，
	// 您必须在 EventHandler 的任何方法中调用它。
	Writev(bs [][]byte) (n int, err error)

	// Flush 将任何缓冲数据写入底层连接，不是并发安全的，
	// 您必须在 EventHandler 的任何方法中调用它。
	Flush() error

	// OutboundBuffered 返回可以从当前缓冲区读取的字节数。
	// 不是并发安全的，您必须在 EventHandler 的任何方法中调用它。
	OutboundBuffered() int

	// AsyncWrite 异步地将字节写入远程，是并发安全的，
	// 您不必在 EventHandler 的任何方法中调用它，
	// 通常您会在单独的 goroutine 中调用它。
	//
	// 注意，对于 UDP，它将同步进行，因此无需调用此异步方法，
	// 我们可能在未来为 UDP 禁用此方法并仅返回 ErrUnsupportedOp，
	// 因此，如果您使用 UDP，请不要依赖此方法来做重要的事情，
	// 只需调用 Conn.Write 来发回您的数据。
	AsyncWrite(buf []byte, callback AsyncCallback) (err error)

	// AsyncWritev 异步地将多个字节切片写入远程，
	// 您不必在 EventHandler 的任何方法中调用它，
	// 通常您会在单独的 goroutine 中调用它。
	AsyncWritev(bs [][]byte, callback AsyncCallback) (err error)
}

// AsyncCallback 是一个回调，在异步函数完成后将被调用。
//
// 注意，当它是 UDP 协议时，参数 xnet.Conn 可能已经被释放，
// 因此不应访问它。
// 此回调将在事件循环中执行，因此它不得阻塞，否则，
// 它会阻塞事件循环。
type AsyncCallback func(c Conn, err error) error

// Socket 是一组操作连接底层文件描述符的函数。
//
// 注意，此接口中的方法对于并发使用是并发安全的，
// 您不必在 EventHandler 的任何方法中调用它们。
type Socket interface {
	// Gfd 返回 socket 的 xgfd。
	// Gfd() xgfd.GFD

	// Fd 返回底层文件描述符。
	Fd() int

	// Dup 返回底层文件描述符的副本。
	// 调用者有责任在完成后关闭 fd。
	// 关闭 c 不影响 fd，关闭 fd 不影响 c。
	//
	// 返回的文件描述符与连接不同。
	// 尝试使用此副本更改原件的属性可能或不可能产生所需的效果。
	Dup() (int, error)

	// SetReadBuffer 设置与连接关联的操作系统的接收缓冲区大小。
	SetReadBuffer(size int) error

	// SetWriteBuffer 设置与连接关联的操作系统的传输缓冲区大小。
	SetWriteBuffer(size int) error

	// SetLinger 设置在仍有数据等待发送或确认的连接上 Close 的行为。
	//
	// 如果 secs < 0（默认），操作系统在后台完成发送数据。
	//
	// 如果 secs == 0，操作系统丢弃任何未发送或未确认的数据。
	//
	// 如果 secs > 0，数据在后台发送，如 secs < 0。在某些操作系统上，经过 sec 秒后，
	// 任何剩余的未发送数据可能会被丢弃。
	SetLinger(secs int) error

	// SetKeepAlivePeriod 告诉操作系统在连接上发送保持活动消息，并设置 TCP 保持活动探测之间的周期。
	SetKeepAlivePeriod(d time.Duration) error

	// SetKeepAlive 启用/禁用 TCP 保持活动，并设置所有套接字选项：
	// TCP_KEEPIDLE、TCP_KEEPINTVL 和 TCP_KEEPCNT。idle 是 TCP_KEEPIDLE 的值，
	// intvl 是 TCP_KEEPINTVL 的值，cnt 是 TCP_KEEPCNT 的值，
	// 当 enabled 为 false 时忽略。
	//
	// 启用 TCP 保持活动后，idle 是连接需要保持空闲的时间（以秒为单位），
	// TCP 开始发送保持活动探测，intvl 是单个保持活动探测之间的时间（以秒为单位）。
	// TCP 在发送 cnt 个探测而没有从对等方获得任何回复后将丢弃连接；
	// 然后套接字被销毁，并触发 OnClose。
	//
	// 如果 idle、intvl 或 cnt 中的一个小于 1，则返回错误。
	SetKeepAlive(enabled bool, idle, intvl time.Duration, cnt int) error

	// SetNoDelay 控制操作系统是否应延迟数据包传输以希望发送更少的数据包（Nagle 算法）。
	// 默认值为 true（无延迟），意味着数据在 Write 后尽快发送。
	SetNoDelay(noDelay bool) error
}

// Runnable 定义在事件循环上执行的通用协议。
// 此接口应以某种方式实现并传递给事件循环，
// 然后事件循环将调用 Run 来执行。
// !!!注意：Run 不得包含任何阻塞操作，如繁重的磁盘或网络 I/O，
// 否则它会阻塞事件循环。
type Runnable interface {
	// Run 即将由事件循环执行。
	Run(ctx context.Context) error
}

// RunnableFunc 是一个适配器，允许将普通函数用作 Runnable。
type RunnableFunc func(ctx context.Context) error

// Run 执行 RunnableFunc 本身。
func (fn RunnableFunc) Run(ctx context.Context) error {
	return fn(ctx)
}

// RegisteredResult 是 Register 调用的结果。
type RegisteredResult struct {
	Conn Conn
	Err  error
}

// EventLoop 提供一组操作事件循环的方法。
type EventLoop interface {
	// Register 连接到给定的地址并将连接注册到当前事件循环，
	// 它是并发安全的。
	Register(ctx context.Context, addr net.Addr) (<-chan RegisteredResult, error)

	// Enroll 类似于 Register，但它接受已建立的 net.Conn 而不是 net.Addr，
	// 它是并发安全的。
	Enroll(ctx context.Context, c net.Conn) (<-chan RegisteredResult, error)

	// Execute 将在未来的某个时间在事件循环上执行给定的 runnable，
	// 它是并发安全的。
	Execute(ctx context.Context, runnable Runnable) error

	// Schedule 类似于 Execute，但它允许您指定何时执行 runnable。
	// 换句话说，当延迟持续时间到达时，runnable 将被执行，
	// 它是并发安全的。
	// 尚未支持，实现此功能。
	Schedule(ctx context.Context, runnable Runnable, delay time.Duration) error

	// Close 关闭属于当前事件循环的给定连接。
	// 它必须在该连接所属的同一个事件循环中调用。
	// 此方法不是并发安全的，您必须在事件循环中调用它。
	Close(Conn) error
}

// Conn 是底层连接的接口。
type Conn interface {
	Reader // Reader 中的所有方法都不是并发安全的。
	Writer // Writer 中有些方法是并发安全的，有些不是。
	Socket // Socket 中的所有方法都是并发安全的。

	// Context 返回用户定义的上下文，不是并发安全的，
	// 您必须在 EventHandler 的任何方法中调用它。
	Context() (ctx any)

	// EventLoop 返回连接所属的事件循环。
	// 返回的 EventLoop 是并发安全的。
	EventLoop() EventLoop

	// SetContext 设置用户定义的上下文，不是并发安全的，
	// 您必须在 EventHandler 的任何方法中调用它。
	SetContext(ctx any)

	// LocalAddr 是连接的本地套接字地址，不是并发安全的，
	// 您必须在 EventHandler 的任何方法中调用它。
	LocalAddr() net.Addr

	// RemoteAddr 是连接的远程地址，不是并发安全的，
	// 您必须在 EventHandler 的任何方法中调用它。
	RemoteAddr() net.Addr

	// Wake 为当前连接触发 OnTraffic 事件，是并发安全的。
	Wake(callback AsyncCallback) error

	// CloseWithCallback 关闭当前连接，是并发安全的。
	// 通常您应该为此方法提供非 nil 回调，
	// 否则您的更好选择是 Close()。
	CloseWithCallback(callback AsyncCallback) error

	// Close 关闭当前连接，实现 net.Conn，是并发安全的。
	Close() error

	// SetDeadline 实现 net.Conn。
	SetDeadline(time.Time) error

	// SetReadDeadline 实现 net.Conn。
	SetReadDeadline(time.Time) error

	// SetWriteDeadline 实现 net.Conn。
	SetWriteDeadline(time.Time) error
}

type (
	// EventHandler 表示 Run 调用的引擎事件回调。
	// 每个事件都有一个 Action 返回值，用于管理连接和引擎的状态。
	EventHandler interface {
		// OnBoot 在引擎准备好接受连接时触发。
		// 参数 engine 包含信息和各种工具。
		OnBoot(eng Engine) (action Action)

		// OnShutdown 在引擎关闭时触发，它在所有事件循环和连接关闭后立即调用。
		OnShutdown(eng Engine)

		// OnOpen 在新连接打开时触发。
		//
		// Conn c 包含有关连接的信息，如其本地和远程地址。
		// 参数 out 是返回值，将发送回远程。
		// 在 OnOpen 中向远程发送大量数据通常不推荐。
		OnOpen(c Conn) (out []byte, action Action)

		// OnClose 在连接关闭时触发。
		// 参数 err 是最后一个已知的连接错误。
		OnClose(c Conn, err error) (action Action)

		// OnTraffic 在套接字从远程接收数据时触发。
		//
		// 另请查看 Reader 和 Writer 接口的注释。
		OnTraffic(c Conn) (action Action)

		// OnTick 在引擎启动后立即触发，并将在延迟返回值的持续时间后再次触发。
		OnTick() (delay time.Duration, action Action)
	}

	// BuiltinEventEngine 是 EventHandler 的内置实现，它为每个方法提供空实现，
	// 当您不打算实现整个 EventHandler 时，可以将其嵌入您的自定义结构体中。
	BuiltinEventEngine struct{}
)

// OnBoot 在引擎准备好接受连接时触发。
// 参数 engine 包含信息和各种工具。
func (*BuiltinEventEngine) OnBoot(_ Engine) (action Action) {
	return
}

// OnShutdown 在引擎关闭时触发，它在所有事件循环和连接关闭后立即调用。
func (*BuiltinEventEngine) OnShutdown(_ Engine) {
}

// OnOpen 在新连接打开时触发。
// 参数 out 是返回值，将发送回远程。
func (*BuiltinEventEngine) OnOpen(_ Conn) (out []byte, action Action) {
	return
}

// OnClose 在连接关闭时触发。
// 参数 err 是最后一个已知的连接错误。
func (*BuiltinEventEngine) OnClose(_ Conn, _ error) (action Action) {
	return
}

// OnTraffic 在套接字从远程接收数据时触发。
func (*BuiltinEventEngine) OnTraffic(_ Conn) (action Action) {
	return
}

// OnTick 在引擎启动后立即触发，并将在延迟返回值的持续时间后再次触发。
func (*BuiltinEventEngine) OnTick() (delay time.Duration, action Action) {
	return
}

// MaxStreamBufferCap 是每个面向流的连接（TCP/Unix）的默认缓冲区大小。
var MaxStreamBufferCap = 64 * 1024 // 64KB

func createListeners(addrs []string, opts ...Option) ([]*listener, *Options, error) {
	options := loadOptions(opts...)

	log.Debugf("default log level is %s", log.LogLevel())

	// Go 程序可以使用的最大操作系统线程数最初设置为 10000，
	// 这也应该是用户可以启动的锁定到 OS 线程的 I/O 事件循环的最大数量。
	if options.LockOSThread && options.NumEventLoop > 10000 {
		log.Errorf("too many event-loops under LockOSThread mode, should be less than 10,000 "+
			"while you are trying to set up %d\n", options.NumEventLoop)
		return nil, nil, xerrors.ErrTooManyEventLoopThreads
	}

	if options.EdgeTriggeredIOChunk > 0 {
		options.EdgeTriggeredIO = true
		options.EdgeTriggeredIOChunk = xmath.CeilToPowerOfTwo(options.EdgeTriggeredIOChunk)
	} else if options.EdgeTriggeredIO {
		options.EdgeTriggeredIOChunk = 1 << 20 // 1MB
	}

	rbc := options.ReadBufferCap
	switch {
	case rbc <= 0:
		options.ReadBufferCap = MaxStreamBufferCap
	case rbc <= xring.DefaultBufferSize:
		options.ReadBufferCap = xring.DefaultBufferSize
	default:
		options.ReadBufferCap = xmath.CeilToPowerOfTwo(rbc)
	}

	wbc := options.WriteBufferCap
	switch {
	case wbc <= 0:
		options.WriteBufferCap = MaxStreamBufferCap
	case wbc <= xring.DefaultBufferSize:
		options.WriteBufferCap = xring.DefaultBufferSize
	default:
		options.WriteBufferCap = xmath.CeilToPowerOfTwo(wbc)
	}

	var hasUDP, hasUnix bool
	for _, addr := range addrs {
		proto, _, err := parseProtoAddr(addr)
		if err != nil {
			return nil, nil, err
		}
		hasUDP = hasUDP || strings.HasPrefix(proto, "udp")
		hasUnix = hasUnix || proto == "unix"
	}

	// SO_REUSEPORT 在各种类 Unix OS 上启用重复地址和端口绑定，
	// 而平台特定不一致：
	// Linux 使用传入连接的负载均衡实现了 SO_REUSEPORT，
	// 而 *BSD 仅实现了绑定到相同地址和端口，这使得在 *BSD 和 Darwin 上为具有多个事件循环的 xnet 启用 SO_REUSEPORT 毫无意义，
	// 因为只有第一个或最后一个事件循环会被不断唤醒以接受传入连接并处理 I/O 事件，而其余事件循环保持空闲。
	// 因此，我们默认在 *BSD 和 Darwin 上禁用 SO_REUSEPORT。
	//
	// 注意，FreeBSD 12 引入了一个名为 SO_REUSEPORT_LB 的新套接字选项，
	// 具有负载均衡能力，它相当于 Linux 的 SO_REUSEPORT。
	// 另请注意，DragonFlyBSD 3.6.0 将 SO_REUSEPORT 扩展到将工作负载分配到可用套接字，这使其与 Linux 的 SO_REUSEPORT 相同。
	goos := runtime.GOOS
	if options.ReusePort &&
		(options.Multicore || options.NumEventLoop > 1) &&
		(goos != "linux" && goos != "dragonfly" && goos != "freebsd") {
		options.ReusePort = false
	}

	// 尽管可以通过 setsockopt() 在 Unix 域套接字上设置 SO_REUSEPORT 而不会报告错误，
	// 但 SO_REUSEPORT 实际上不支持 AF_UNIX 的套接字。因此，我们避免在 Unix 域套接字上设置它。
	// 从这个提交 https://git.kernel.org/pub/scm/linux/kernel/git/netdev/net.git/commit/?id=5b0af621c3f6 开始，
	// 在 Linux 上尝试在 AF_UNIX 套接字上设置 SO_REUSEPORT 时将返回 EOPNOTSUPP。
	// 因此，我们在所有类 Unix 平台上避免在 Unix 域套接字上设置它，以保持此行为一致。
	if options.ReusePort && hasUnix {
		options.ReusePort = false
	}

	// 如果列表中有 UDP 地址，我们别无选择，只能启用 SO_REUSEPORT，
	// 也默认禁用 UDP 的边缘触发 I/O。
	if hasUDP {
		options.ReusePort = true
		options.EdgeTriggeredIO = false
	}

	listeners := make([]*listener, len(addrs))
	for i, a := range addrs {
		proto, addr, err := parseProtoAddr(a)
		if err != nil {
			return nil, nil, err
		}
		ln, err := initListener(proto, addr, options)
		if err != nil {
			return nil, nil, err
		}
		listeners[i] = ln
	}

	return listeners, options, nil
}

// Run 开始在指定地址上处理事件。
//
// 地址应使用方案前缀并格式化为
// `tcp://192.168.0.10:9851` 或 `unix://socket`。
// 有效网络方案：
//
//	tcp   - 绑定到 IPv4 和 IPv6
//	tcp4  - IPv4
//	tcp6  - IPv6
//	udp   - 绑定到 IPv4 和 IPv6
//	udp4  - IPv4
//	udp6  - IPv6
//	unix  - Unix 域套接字
//
// 如果未指定，则假定为 "tcp" 网络方案。
func Run(eventHandler EventHandler, protoAddr string, opts ...Option) error {
	listeners, options, err := createListeners([]string{protoAddr}, opts...)
	if err != nil {
		return err
	}
	defer func() {
		for _, ln := range listeners {
			ln.close()
		}
	}()
	return run(eventHandler, listeners, options, []string{protoAddr})
}

// Rotate 类似于 Run 但接受多个网络地址。
func Rotate(eventHandler EventHandler, addrs []string, opts ...Option) error {
	listeners, options, err := createListeners(addrs, opts...)
	if err != nil {
		return err
	}
	defer func() {
		for _, ln := range listeners {
			ln.close()
		}
	}()
	return run(eventHandler, listeners, options, addrs)
}

var (
	allEngines sync.Map

	// shutdownPollInterval 是我们在 xnet.Stop() 期间轮询检查引擎是否已关闭的频率。
	shutdownPollInterval = 500 * time.Millisecond
)

// Stop 优雅地关闭引擎而不中断任何活跃的事件循环，
// 它无限期等待连接和事件循环关闭，然后关闭。
//
// 已弃用：全局 Stop 仅关闭与之前引擎具有相同协议和 IP:Port 的最后一个注册引擎，
// 如果您在启用 WithReuseAddr(true) 和 WithReusePort(true) 的情况下多次调用 xnet.Run 使用相同的协议和 IP:Port，
// 这可能导致引擎泄漏。使用 Engine.Stop 代替。
func Stop(ctx context.Context, protoAddr string) error {
	var eng *engine
	if s, ok := allEngines.Load(protoAddr); ok {
		eng = s.(*engine)
		eng.shutdown(nil)
		defer allEngines.Delete(protoAddr)
	} else {
		return xerrors.ErrEngineInShutdown
	}

	if eng.isShutdown() {
		return xerrors.ErrEngineInShutdown
	}

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if eng.isShutdown() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func parseProtoAddr(protoAddr string) (string, string, error) {
	// 对地址中的 "%" 做百分号编码，避免 url.Parse 报错。
	// 例如：udp://[ff02::3%lo0]:9991
	protoAddr = strings.ReplaceAll(protoAddr, "%", "%25")

	if runtime.GOOS == "windows" {
		if strings.HasPrefix(protoAddr, "unix://") {
			parts := strings.SplitN(protoAddr, "://", 2)
			if parts[1] == "" {
				return "", "", xerrors.ErrInvalidNetworkAddress
			}
			return parts[0], parts[1], nil
		}
	}

	u, err := url.Parse(protoAddr)
	if err != nil {
		return "", "", err
	}

	switch u.Scheme {
	case "":
		return "", "", xerrors.ErrInvalidNetworkAddress
	case "tcp", "tcp4", "tcp6", "udp", "udp4", "udp6":
		if u.Host == "" || u.Path != "" {
			return "", "", xerrors.ErrInvalidNetworkAddress
		}
		return u.Scheme, u.Host, nil
	case "unix":
		hostPath := path.Join(u.Host, u.Path)
		if hostPath == "" {
			return "", "", xerrors.ErrInvalidNetworkAddress
		}
		return u.Scheme, hostPath, nil
	default:
		return "", "", xerrors.ErrUnsupportedProtocol
	}
}

func determineEventLoops(opts *Options) int {
	numEventLoop := 1

	if opts.Multicore {
		numEventLoop = runtime.NumCPU()
	}
	if opts.NumEventLoop > 0 {
		numEventLoop = opts.NumEventLoop
	}
	if numEventLoop > xgfd.EventLoopIndexMax {
		numEventLoop = xgfd.EventLoopIndexMax
	}

	return numEventLoop
}
