package packet

type Message struct {
	Seq       int32  // 序列号
	GameID    int32  // 游戏ID 0 表示到大厅
	MessageID int32  // 消息ID
	Buffer    []byte // 消息内容
}
