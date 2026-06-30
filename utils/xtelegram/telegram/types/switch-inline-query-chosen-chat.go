package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// SwitchInlineQueryChosenChat 表示在所选聊天中将当前用户切换到内联模式的内联按钮，
// 带有可选的默认内联查询。
type SwitchInlineQueryChosenChat struct {
	Query             string `json:"query,omitempty"`               // 可选。要插入输入字段中的默认内联查询。如果留空，则仅插入机器人的用户名
	AllowUserChats    bool   `json:"allow_user_chats,omitempty"`    // 可选。如果可以选择与用户的私聊，则为 true
	AllowBotChats     bool   `json:"allow_bot_chats,omitempty"`     // 可选。如果可以选择与机器人的私聊，则为 true
	AllowGroupChats   bool   `json:"allow_group_chats,omitempty"`   // 可选。如果可以选择群组和超级群组聊天，则为 true
	AllowChannelChats bool   `json:"allow_channel_chats,omitempty"` // 可选。如果可以选择频道聊天，则为 true
}

func (j SwitchInlineQueryChosenChat) Value() (driver.Value, error) {
	if j.IsNull() {
		return nil, nil
	}
	byteData, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(byteData), nil
}
func (j *SwitchInlineQueryChosenChat) Scan(value any) error {
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
func (j SwitchInlineQueryChosenChat) IsNull() bool {
	return j.Query == ""
}
func (j SwitchInlineQueryChosenChat) Clone() *SwitchInlineQueryChosenChat {
	if j.IsNull() {
		return nil
	}
	return &SwitchInlineQueryChosenChat{
		Query:             j.Query,             // 可选。要插入输入字段中的默认内联查询。如果留空，则仅插入机器人的用户名
		AllowUserChats:    j.AllowUserChats,    // 可选。如果可以选择与用户的私聊，则为 true
		AllowBotChats:     j.AllowBotChats,     // 可选。如果可以选择与机器人的私聊，则为 true
		AllowGroupChats:   j.AllowGroupChats,   // 可选。如果可以选择群组和超级群组聊天，则为 true
		AllowChannelChats: j.AllowChannelChats, // 可选。如果可以选择频道聊天，则为 true
	}
}
