package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// KeyboardButtonPollType 表示允许在相应按钮被按下时创建和发送的投票类型。
type KeyboardButtonPollType struct {
	Type string `json:"type"` // 可选。如果传递 quiz，用户将仅被允许创建测验模式的投票。如果传递 regular，则仅允许常规投票。否则，用户将被允许创建任何类型的投票。
}

func (j KeyboardButtonPollType) Value() (driver.Value, error) {
	if j.IsNull() {
		return nil, nil
	}
	byteData, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(byteData), nil
}
func (j *KeyboardButtonPollType) Scan(value any) error {
	if value == nil {
		j = nil
		return nil
	}
	s, ok := value.([]byte)
	if !ok {
		return errors.New("invalid Scan Source")
	}
	return json.Unmarshal(s, j)
}
func (j KeyboardButtonPollType) IsNull() bool {
	return j.Type == ""
}
func (j KeyboardButtonPollType) Clone() *KeyboardButtonPollType {
	if j.IsNull() {
		return nil
	}
	return &KeyboardButtonPollType{
		Type: j.Type, // 将在 UsersShared 对象中接收到的请求的有符号 32 位标识符。必须在消息中唯一。

	}
}
