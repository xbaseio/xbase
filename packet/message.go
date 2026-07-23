package packet

type Message struct {
	Seq       int32  // 序列号
	GameID    int32  // 游戏ID；0 表示 Gate，1 表示大厅，2+ 表示具体游戏，网关按此字段分流
	MessageID int32  // 消息ID；节点内按此字段匹配业务处理器
	Buffer    []byte // 消息内容
}
