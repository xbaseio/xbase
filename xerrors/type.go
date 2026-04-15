package xerrors

import "github.com/xbaseio/xbase/codes"

// -----------------------------------------------------------------------------
// 内部辅助方法
// -----------------------------------------------------------------------------
//
// 目的：
// 1. 减少重复的 New("xxx") / NewCode(..., "xxx")
// 2. 让错误定义更短、更整齐
// 3. 后续如果要统一包装前缀、日志标记、国际化，也更容易扩展
//

// errText 创建一个普通文本错误。
func errText(msg string) *Error {
	return New(msg)
}

// errCode 创建一个带业务错误码的错误。
func errCode(code *codes.Code, msg string) *Error {
	return NewCode(code, msg)
}

// xnetErr 创建一个带 xnet 前缀的内部网络层错误。
// 这样可以避免反复写 "xnet: ..."
func xnetErr(msg string) *Error {
	return New("xnet: " + msg)
}

// -----------------------------------------------------------------------------
// 通用错误
// -----------------------------------------------------------------------------

var (
	// ErrNil 表示空值错误。
	ErrNil = errText("nil")

	// ErrInvalidArgument 表示参数不合法。
	ErrInvalidArgument = errCode(codes.InvalidArgument, "invalid argument")

	// ErrInvalidPointer 表示传入了非法指针。
	ErrInvalidPointer = errText("invalid pointer")

	// ErrInvalidFormat 表示数据格式非法。
	ErrInvalidFormat = errText("invalid format")

	// ErrUnexpectedEOF 表示读取过程中遇到了非预期的 EOF。
	ErrUnexpectedEOF = errText("unexpected EOF")

	// ErrUnknownError 表示未知错误。
	ErrUnknownError = errCode(codes.Unknown, "unknown error")

	// ErrDeadlineExceeded 表示处理超时。
	ErrDeadlineExceeded = errCode(codes.DeadlineExceeded, "deadline exceeded")

	// ErrIllegalRequest 表示非法请求。
	ErrIllegalRequest = errCode(codes.IllegalRequest, "illegal request")

	// ErrIllegalOperation 表示非法操作。
	ErrIllegalOperation = errText("illegal operation")

	// ErrNoOperationPermission 表示没有操作权限。
	ErrNoOperationPermission = errText("no operation permission")
)

// -----------------------------------------------------------------------------
// 标识 / 输入 / 协议相关错误
// -----------------------------------------------------------------------------

var (
	// ErrInvalidGID 表示 gate id 非法。
	ErrInvalidGID = errText("invalid gate id")

	// ErrInvalidNID 表示 node id 非法。
	ErrInvalidNID = errText("invalid node id")

	// ErrInvalidMessage 表示消息体非法。
	ErrInvalidMessage = errText("invalid message")

	// ErrInvalidReader 表示 reader 非法。
	ErrInvalidReader = errText("invalid reader")

	// ErrInvalidDecoder 表示 decoder 非法。
	ErrInvalidDecoder = errText("invalid decoder")

	// ErrInvalidScanner 表示 scanner 非法。
	ErrInvalidScanner = errText("invalid scanner")

	// ErrReceiveTargetEmpty 表示接收目标为空。
	ErrReceiveTargetEmpty = errText("the receive target is empty")

	// ErrSeqOverflow 表示序列号溢出。
	ErrSeqOverflow = errText("seq overflow")

	// ErrRouteOverflow 表示路由数量或路由索引溢出。
	ErrRouteOverflow = errText("route overflow")

	// ErrMessageTooLarge 表示消息体过大。
	ErrMessageTooLarge = errText("message too large")
)

// -----------------------------------------------------------------------------
// 查找 / 路由 / 事件相关错误
// -----------------------------------------------------------------------------

var (
	// ErrNotFoundSession 表示未找到会话。
	ErrNotFoundSession = errText("not found session")

	// ErrInvalidSessionKind 表示会话类型非法。
	ErrInvalidSessionKind = errText("invalid session kind")

	// ErrNotFoundRoute 表示未找到路由。
	ErrNotFoundRoute = errText("not found route")

	// ErrNotFoundEvent 表示未找到事件。
	ErrNotFoundEvent = errText("not found event")

	// ErrNotFoundEndpoint 表示未找到 endpoint。
	ErrNotFoundEndpoint = errText("not found endpoint")

	// ErrNotFoundUserLocation 表示未找到用户位置。
	ErrNotFoundUserLocation = errText("not found user's location")

	// ErrNotFoundLocator 表示未找到定位器。
	ErrNotFoundLocator = errText("not found locator")

	// ErrNotFoundServiceAddress 表示未找到服务地址。
	ErrNotFoundServiceAddress = errText("not found service address")

	// ErrNotFoundIPAddress 表示未找到 IP 地址。
	ErrNotFoundIPAddress = errText("not found ip address")
)

// -----------------------------------------------------------------------------
// 客户端 / 连接 / 服务状态相关错误
// -----------------------------------------------------------------------------

var (
	// ErrClientShut 表示客户端正在关闭或已停止工作。
	ErrClientShut = errText("client is shut")

	// ErrClientClosed 表示客户端已关闭。
	ErrClientClosed = errText("client is closed")

	// ErrServerClosed 表示服务端已关闭。
	ErrServerClosed = errText("server is closed")

	// ErrConnectionOpened 表示连接已打开。
	ErrConnectionOpened = errText("connection is opened")

	// ErrConnectionHanged 表示连接已挂起。
	ErrConnectionHanged = errText("connection is hanged")

	// ErrConnectionClosed 表示连接已关闭。
	ErrConnectionClosed = errText("connection is closed")

	// ErrConnectionNotOpened 表示连接尚未打开。
	ErrConnectionNotOpened = errText("connection is not opened")

	// ErrConnectionNotHanged 表示连接并未处于挂起状态。
	ErrConnectionNotHanged = errText("connection is not hanged")

	// ErrTooManyConnection 表示连接数过多。
	ErrTooManyConnection = errText("too many connection")

	// ErrServiceRegisterFailed 表示服务注册失败。
	ErrServiceRegisterFailed = errText("service register failed")

	// ErrServiceDeregisterFailed 表示服务注销失败。
	ErrServiceDeregisterFailed = errText("service deregister failed")
)

// -----------------------------------------------------------------------------
// 配置 / 组件依赖相关错误
// -----------------------------------------------------------------------------

var (
	// ErrInvalidConfigContent 表示配置内容非法。
	ErrInvalidConfigContent = errText("invalid config content")

	// ErrNotFoundConfigSource 表示未找到配置来源。
	ErrNotFoundConfigSource = errText("not found config source")

	// ErrMissingTransporter 表示缺少 transporter 组件。
	ErrMissingTransporter = errText("missing transporter")

	// ErrMissingDiscovery 表示缺少服务发现组件。
	ErrMissingDiscovery = errText("missing discovery")

	// ErrMissingResolver 表示缺少 resolver 组件。
	ErrMissingResolver = errText("missing resolver")

	// ErrMissingDispatchStrategy 表示缺少分发策略。
	ErrMissingDispatchStrategy = errText("missing dispatch strategy")

	// ErrMissingCacheInstance 表示缺少缓存实例。
	ErrMissingCacheInstance = errText("missing cache instance")

	// ErrMissingEventbusInstance 表示缺少 eventbus 实例。
	ErrMissingEventbusInstance = errText("missing eventbus instance")

	// ErrInvalidServiceDesc 表示服务描述不合法。
	ErrInvalidServiceDesc = errText("invalid service desc")
)

// -----------------------------------------------------------------------------
// Actor / 路由绑定相关错误
// -----------------------------------------------------------------------------

var (
	// ErrActorExists 表示 actor 已存在。
	ErrActorExists = errText("actor exists")

	// ErrUnregisterRoute 表示路由尚未注册。
	ErrUnregisterRoute = errText("unregistered route")

	// ErrNotBindActor 表示尚未绑定 actor。
	ErrNotBindActor = errText("not bind actor")

	// ErrNotFoundActor 表示未找到 actor。
	ErrNotFoundActor = errText("not found actor")

	// ErrSyncerClosed 表示同步器已关闭。
	ErrSyncerClosed = errText("syncer is closed")
)

// -----------------------------------------------------------------------------
// 安全 / 加密相关错误
// -----------------------------------------------------------------------------

var (
	// ErrInvalidPublicKey 表示公钥非法。
	ErrInvalidPublicKey = errText("invalid public key")

	// ErrInvalidPrivateKey 表示私钥非法。
	ErrInvalidPrivateKey = errText("invalid private key")

	// ErrInvalidSignature 表示签名非法。
	ErrInvalidSignature = errText("invalid signature")

	// ErrInvalidCertFile 表示证书文件非法。
	ErrInvalidCertFile = errText("invalid cert file")
)

// -----------------------------------------------------------------------------
// xnet 内部错误
// -----------------------------------------------------------------------------

var (
	// ErrEmptyEngine 表示 xnet 内部 engine 为空。
	ErrEmptyEngine = xnetErr("the internal engine is empty")

	// ErrEngineShutdown 表示服务正在关闭。
	ErrEngineShutdown = xnetErr("server is going to be shutdown")

	// ErrEngineInShutdown 表示重复执行关闭流程。
	ErrEngineInShutdown = xnetErr("server is already in shutdown")

	// ErrAcceptSocket 表示 accept 新连接失败。
	ErrAcceptSocket = xnetErr("accept a new connection error")

	// ErrTooManyEventLoopThreads 表示 LockOSThread 模式下 event-loop 线程过多。
	ErrTooManyEventLoopThreads = xnetErr("too many event-loops under LockOSThread mode")

	// ErrUnsupportedProtocol 表示协议不受支持。
	ErrUnsupportedProtocol = xnetErr("only unix, tcp/tcp4/tcp6, udp/udp4/udp6 are supported")

	// ErrUnsupportedTCPProtocol 表示 TCP 协议不受支持。
	ErrUnsupportedTCPProtocol = xnetErr("only tcp/tcp4/tcp6 are supported")

	// ErrUnsupportedUDPProtocol 表示 UDP 协议不受支持。
	ErrUnsupportedUDPProtocol = xnetErr("only udp/udp4/udp6 are supported")

	// ErrUnsupportedUDSProtocol 表示 Unix Domain Socket 协议不受支持。
	ErrUnsupportedUDSProtocol = xnetErr("only unix is supported")

	// ErrUnsupportedOp 表示当前操作不支持或尚未实现。
	ErrUnsupportedOp = xnetErr("unsupported operation")

	// ErrNegativeSize 表示传入了负数大小。
	ErrNegativeSize = xnetErr("negative size is not allowed")

	// ErrNoIPv4AddressOnInterface 表示接口上没有配置 IPv4 地址。
	ErrNoIPv4AddressOnInterface = xnetErr("no IPv4 address on interface")

	// ErrInvalidNetworkAddress 表示网络地址非法。
	ErrInvalidNetworkAddress = xnetErr("invalid network address")

	// ErrInvalidNetConn 表示 net.Conn 为空。
	ErrInvalidNetConn = xnetErr("the net.Conn is empty")

	// ErrNilRunnable 表示 runnable 为空。
	ErrNilRunnable = xnetErr("nil runnable is not allowed")
)
