package packet

type Message struct {
	Seq       int32  // 序列号
	GameID    int32  // 游戏ID；0 表示大厅，非 0 表示具体游戏服，网关按此字段选择目标节点
	MessageID int32  // 消息ID；节点内按此字段匹配业务处理器
	Buffer    []byte // 消息内容
}
