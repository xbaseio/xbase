package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// KeyboardButtonRequestUsers 定义用于请求合适用户的标准。
// 当相应的按钮被按下时，所选用户的标识符将与机器人共享。
type KeyboardButtonRequestUsers struct {
	RequestId       int  `json:"request_id"`                 // 将在 UsersShared 对象中接收到的请求的有符号 32 位标识符。必须在消息中唯一。
	UserIsBot       bool `json:"user_is_bot,omitempty"`      // 可选。传递 true 以请求机器人，传递 false 以请求常规用户。如果未指定，则不应用任何额外限制。
	UserIsPremium   bool `json:"user_is_premium,omitempty"`  // 可选。传递 true 以请求高级用户，传递 false 以请求非高级用户。如果未指定，则不应用任何额外限制。
	MaxQuantity     int  `json:"max_quantity,omitempty"`     // 可选。要选择的最大用户数量；1-10。默认值为 1。
	RequestName     bool `json:"request_name,omitempty"`     // 可选。传递 true 以请求用户的名字和姓氏
	RequestUsername bool `json:"request_username,omitempty"` // 可选。传递 true 以请求用户的用户名
	RequestPhoto    bool `json:"request_photo,omitempty"`    // 可选。传递 true 以请求用户的照片
}

func (j KeyboardButtonRequestUsers) Value() (driver.Value, error) {
	if j.IsNull() {
		return nil, nil
	}
	byteData, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(byteData), nil
}
func (j *KeyboardButtonRequestUsers) Scan(value any) error {
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
func (j KeyboardButtonRequestUsers) IsNull() bool {
	return j.RequestId == 0
}

func (j KeyboardButtonRequestUsers) Clone() *KeyboardButtonRequestUsers {
	if j.IsNull() {
		return nil
	}
	return &KeyboardButtonRequestUsers{
		RequestId:       j.RequestId,       // 将在 UsersShared 对象中接收到的请求的有符号 32 位标识符。必须在消息中唯一。
		UserIsBot:       j.UserIsBot,       // 可选。传递 true 以请求机器人，传递 false 以请求常规用户。如果未指定，则不应用任何额外限制。
		UserIsPremium:   j.UserIsPremium,   // 可选。传递 true 以请求高级用户，传递 false 以请求非高级用户。如果未指定，则不应用任何额外限制。
		MaxQuantity:     j.MaxQuantity,     // 可选。要选择的最大用户数量；1-10。默认值为 1。
		RequestName:     j.RequestName,     // 可选。传递 true 以请求用户的名字和姓氏
		RequestUsername: j.RequestUsername, // 可选。传递 true 以请求用户的用户名
		RequestPhoto:    j.RequestPhoto,    // 可选。传递 true 以请求用户的照片
	}

}
