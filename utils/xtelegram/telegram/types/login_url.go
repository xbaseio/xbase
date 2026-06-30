package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// LoginURL 表示用于自动授权用户的内联键盘按钮的参数。
// 作为 Telegram 登录小部件的一个很好的替代方案，当用户来自 Telegram 时。
// 用户只需点击按钮并确认他们希望登录：
// Telegram 应用程序从版本 5.7 开始支持这些按钮。
// 示例机器人：@discuss-bot
type LoginURL struct {
	URL                string `json:"url"`                            // 按下按钮时打开的 HTTP URL，用户授权数据将添加到查询字符串中。如果用户拒绝提供授权数据，则将打开不带用户信息的原始 URL。添加的数据与接收授权数据中描述的数据相同。注意：您必须始终检查接收数据的哈希，以验证身份验证和数据的完整性，如检查授权中所述。
	ForwardText        string `json:"forward_text,omitempty"`         // 可选。转发消息中按钮的新文本。
	BotUsername        string `json:"bot_username,omitempty"`         // 可选。将用于用户授权的机器人的用户名。有关详细信息，请参见设置机器人。如果未指定，将假定当前机器人的用户名。hurl 的域名必须与链接到机器人的域名相同。有关详细信息，请参见将您的域名链接到机器人。
	RequestWriteAccess bool   `json:"request_write_access,omitempty"` // 可选。传递 true 以请求机器人向用户发送消息的权限。
}

func (j LoginURL) Value() (driver.Value, error) {
	if j.IsNull() {
		return nil, nil
	}
	byteData, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(byteData), nil
}
func (j *LoginURL) Scan(value any) error {
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
func (j LoginURL) IsNull() bool {
	return j.URL == ""
}
func (j LoginURL) Clone() *LoginURL {
	if j.IsNull() {
		return nil
	}
	return &LoginURL{
		URL:                j.URL,                // 按下按钮时打开的 HTTP URL，用户授权数据将添加到查询字符串中。如果用户拒绝提供授权数据，则将打开不带用户信息的原始 URL。添加的数据与接收授权数据中描述的数据相同。注意：您必须始终检查接收数据的哈希，以验证身份验证和数据的完整性，如检查授权中所述。
		ForwardText:        j.ForwardText,        // 可选。转发消息中按钮的新文本。
		BotUsername:        j.BotUsername,        // 可选。将用于用户授权的机器人的用户名。有关详细信息，请参见设置机器人。如果未指定，将假定当前机器人的用户名。hurl 的域名必须与链接到机器人的域名相同。有关详细信息，请参见将您的域名链接到机器人。
		RequestWriteAccess: j.RequestWriteAccess, // 可选。传递 true 以请求机器人向用户发送消息的权限。
	}
}
