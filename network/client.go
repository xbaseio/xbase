package network

type Client interface {
	// Dial 拨号连接
	Dial(addr ...string) (Conn, error)
	// Protocol 协议
	Protocol() string
	// OnConnect 监听连接打开
	OnConnect(handler ConnectHandler)
	// OnReceive 监听接收消息
	OnReceive(handler ReceiveHandler)
	// OnDisconnect 监听连接断开
	OnDisconnect(handler DisconnectHandler)
}
