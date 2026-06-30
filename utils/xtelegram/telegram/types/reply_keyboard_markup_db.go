package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type ReplyKeyboardMarkupDb struct {
	IsPersistent          bool   `json:"is_persistent,omitempty"`           // 可选。请求客户端在常规键盘隐藏时始终显示此键盘。默认值为 false，此时自定义键盘可以被隐藏，并通过键盘图标打开。
	ResizeKeyboard        bool   `json:"resize_keyboard,omitempty"`         // 可选。请求客户端根据最佳适配垂直调整键盘大小（例如，如果只有两行按钮，则使键盘变小）。默认值为 false，此时自定义键盘的高度始终与应用程序的标准键盘相同。
	OneTimeKeyboard       bool   `json:"one_time_keyboard,omitempty"`       // 可选。请求客户端在使用此键盘后隐藏键盘。键盘仍然可用，但客户端将自动在聊天中显示常规字母键盘 - 用户可以在输入字段中按特殊按钮再次查看自定义键盘。默认值为 false。
	InputFieldPlaceholder string `json:"input_field_placeholder,omitempty"` // 可选。当键盘处于活动状态时，在输入字段中显示的占位符；1-64 个字符
	Selective             bool   `json:"selective,omitempty"`               // 可选。如果只想对特定用户显示此键盘，请使用此参数。目标：1）在消息对象的文本中提到的用户；2）如果机器人的消息是回复（具有 reply_to_message_id），则是原始消息的发送者。示例：用户请求更改机器人的语言，机器人回复请求并显示选择新语言的键盘。其他用户在群组中看不到此键盘。
}

func (j ReplyKeyboardMarkupDb) Value() (driver.Value, error) {
	byteData, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(byteData), nil
}
func (j *ReplyKeyboardMarkupDb) Scan(value any) error {
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
func (j ReplyKeyboardMarkupDb) IsNull() bool {
	return j.InputFieldPlaceholder == "" && (!j.IsPersistent) && (!j.ResizeKeyboard) && (!j.OneTimeKeyboard) && (!j.Selective)
}
func (j ReplyKeyboardMarkupDb) Clone() *ReplyKeyboardMarkupDb {
	if j.IsNull() {
		return nil
	}
	return &ReplyKeyboardMarkupDb{
		IsPersistent:          j.IsPersistent,          // 可选。请求客户端在常规键盘隐藏时始终显示此键盘。默认值为 false，此时自定义键盘可以被隐藏，并通过键盘图标打开。
		ResizeKeyboard:        j.ResizeKeyboard,        // 可选。请求客户端根据最佳适配垂直调整键盘大小（例如，如果只有两行按钮，则使键盘变小）。默认值为 false，此时自定义键盘的高度始终与应用程序的标准键盘相同。
		OneTimeKeyboard:       j.OneTimeKeyboard,       // 可选。请求客户端在使用此键盘后隐藏键盘。键盘仍然可用，但客户端将自动在聊天中显示常规字母键盘 - 用户可以在输入字段中按特殊按钮再次查看自定义键盘。默认值为 false。
		InputFieldPlaceholder: j.InputFieldPlaceholder, // 可选。当键盘处于活动状态时，在输入字段中显示的占位符；1-64 个字符
		Selective:             j.Selective,             // 可选。如果只想对特定用户显示此键盘，请使用此参数。目标：1）在消息对象的文本中提到的用户；2）如果机器人的消息是回复（具有 reply_to_message_id），则是原始消息的发送者。示例：用户请求更改机器人的语言，机器人回复请求并显示选择新语言的键盘。其他用户在群组中看不到此键盘。

	}
}
