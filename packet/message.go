package packet

type Message struct {
	Seq       int32  // 序列号
	NodeID    int32  // 节点ID
	MessageID int32  // 消息ID
	Buffer    []byte // 消息内容
}
