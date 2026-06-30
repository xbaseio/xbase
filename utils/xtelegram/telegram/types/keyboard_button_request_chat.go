package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// KeyboardButtonRequestChat 定义用于请求合适聊天的标准。
// 当相应的按钮被按下时，所选聊天的标识符将与机器人共享。
type KeyboardButtonRequestChat struct {
	RequestID               int                      `json:"request_id"`                          // 将在 ChatShared 对象中接收到的请求的有符号 32 位标识符。必须在消息中唯一。
	ChatIsChannel           bool                     `json:"chat_is_channel"`                     // 传递 true 以请求频道聊天，传递 false 以请求群组或超级群组聊天。
	ChatIsForum             bool                     `json:"chat_is_forum,omitempty"`             // 可选。传递 true 以请求论坛超级群组，传递 false 以请求非论坛聊天。如果未指定，则不应用任何额外限制。
	ChatHasUsername         bool                     `json:"chat_has_username,omitempty"`         // 可选。传递 true 以请求具有用户名的超级群组或频道，传递 false 以请求没有用户名的聊天。如果未指定，则不应用任何额外限制。
	ChatIsCreated           bool                     `json:"chat_is_created,omitempty"`           // 可选。传递 true 以请求由用户拥有的聊天。否则，不应用任何额外限制。
	UserAdministratorRights *ChatAdministratorRights `json:"user_administrator_rights,omitempty"` // 可选。一个 JSON 序列化的对象，列出用户在聊天中所需的管理员权限。权限必须是 bot_administrator_rights 的超集。如果未指定，则不应用任何额外限制。
	BotAdministratorRights  *ChatAdministratorRights `json:"bot_administrator_rights,omitempty"`  // 可选。一个 JSON 序列化的对象，列出机器人在聊天中所需的管理员权限。权限必须是 user_administrator_rights 的子集。如果未指定，则不应用任何额外限制。
	BotIsMember             bool                     `json:"bot_is_member,omitempty"`             // 可选。传递 true 以请求与机器人作为成员的聊天。否则，不应用任何额外限制。
	RequestTitle            bool                     `json:"request_title,omitempty"`             // 可选。传递 true 以请求聊天标题。
	RequestUsername         bool                     `json:"request_username,omitempty"`          // 可选。传递 true 以请求聊天的用户名。
	RequestPhoto            bool                     `json:"request_photo,omitempty"`             // 可选。传递 true 以请求聊天的照片。
}

func (j KeyboardButtonRequestChat) Value() (driver.Value, error) {
	if j.IsNull() {
		return nil, nil
	}
	byteData, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	return string(byteData), nil
}
func (j *KeyboardButtonRequestChat) Scan(value any) error {
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
func (j KeyboardButtonRequestChat) IsNull() bool {
	return j.RequestID == 0
}

func (j KeyboardButtonRequestChat) Clone() *KeyboardButtonRequestChat {
	if j.IsNull() {
		return nil
	}
	return &KeyboardButtonRequestChat{
		RequestID:       j.RequestID,       // 将在 ChatShared 对象中接收到的请求的有符号 32 位标识符。必须在消息中唯一。
		ChatIsChannel:   j.ChatIsChannel,   // 传递 true 以请求频道聊天，传递 false 以请求群组或超级群组聊天。
		ChatIsForum:     j.ChatIsForum,     // 可选。传递 true 以请求论坛超级群组，传递 false 以请求非论坛聊天。如果未指定，则不应用任何额外限制。
		ChatHasUsername: j.ChatHasUsername, // 可选。传递 true 以请求具有用户名的超级群组或频道，传递 false 以请求没有用户名的聊天。如果未指定，则不应用任何额外限制。
		ChatIsCreated:   j.ChatIsCreated,   // 可选。传递 true 以请求由用户拥有的聊天。否则，不应用任何额外限制。
		UserAdministratorRights: &ChatAdministratorRights{
			IsAnonymous:         j.UserAdministratorRights.IsAnonymous,         // 如果用户在聊天中的存在是隐藏的，则为真
			CanManageChat:       j.UserAdministratorRights.CanManageChat,       // 如果管理员可以访问聊天事件日志、聊天统计、频道消息统计、查看频道成员、查看超级群组中的匿名管理员并忽略慢模式，则为真。由任何其他管理员权限隐含。
			CanDeleteMessages:   j.UserAdministratorRights.CanDeleteMessages,   // 如果管理员可以删除其他用户的消息，则为真
			CanManageVideoChats: j.UserAdministratorRights.CanManageVideoChats, // 如果管理员可以管理视频聊天，则为真
			CanRestrictMembers:  j.UserAdministratorRights.CanRestrictMembers,  // 如果管理员可以限制、禁止或解除禁止聊天成员，则为真
			CanPromoteMembers:   j.UserAdministratorRights.CanPromoteMembers,   // 如果管理员可以添加新的管理员并拥有部分自己的权限，或者直接或间接地降低他提升的管理员的权限，则为真
			CanChangeInfo:       j.UserAdministratorRights.CanChangeInfo,       // 如果用户被允许更改聊天标题、照片和其他设置，则为真
			CanInviteUsers:      j.UserAdministratorRights.CanInviteUsers,      // 如果用户被允许邀请新用户加入聊天，则为真
			CanPostMessages:     j.UserAdministratorRights.CanPostMessages,     // 可选。如果管理员可以在频道中发帖，则为真；仅适用于频道
			CanEditMessages:     j.UserAdministratorRights.CanEditMessages,     // 可选。如果管理员可以编辑其他用户的消息并可以固定消息，则为真；仅适用于频道
			CanPinMessages:      j.UserAdministratorRights.CanPinMessages,      // 可选。如果用户被允许固定消息，则为真；仅适用于群组和超级群组
			CanPostStories:      j.UserAdministratorRights.CanPostStories,      // 可选。如果管理员可以在频道中发布故事，则为真；仅适用于频道
			CanEditStories:      j.UserAdministratorRights.CanEditStories,      // 可选。如果管理员可以编辑其他用户发布的故事，则为真；仅适用于频道
			CanDeleteStories:    j.UserAdministratorRights.CanDeleteStories,    // 可选。如果管理员可以删除其他用户发布的故事，则为真；仅适用于频道
			CanManageTopics:     j.UserAdministratorRights.CanManageTopics,     // 可选。如果用户被允许创建、重命名、关闭和重新打开论坛主题，则为真；仅适用于超级群组
		}, // 可选。一个 JSON 序列化的对象，列出用户在聊天中所需的管理员权限。权限必须是 bot_administrator_rights 的超集。如果未指定，则不应用任何额外限制。
		BotAdministratorRights: &ChatAdministratorRights{
			IsAnonymous:         j.BotAdministratorRights.IsAnonymous,         // 如果用户在聊天中的存在是隐藏的，则为真
			CanManageChat:       j.BotAdministratorRights.CanManageChat,       // 如果管理员可以访问聊天事件日志、聊天统计、频道消息统计、查看频道成员、查看超级群组中的匿名管理员并忽略慢模式，则为真。由任何其他管理员权限隐含。
			CanDeleteMessages:   j.BotAdministratorRights.CanDeleteMessages,   // 如果管理员可以删除其他用户的消息，则为真
			CanManageVideoChats: j.BotAdministratorRights.CanManageVideoChats, // 如果管理员可以管理视频聊天，则为真
			CanRestrictMembers:  j.BotAdministratorRights.CanRestrictMembers,  // 如果管理员可以限制、禁止或解除禁止聊天成员，则为真
			CanPromoteMembers:   j.BotAdministratorRights.CanPromoteMembers,   // 如果管理员可以添加新的管理员并拥有部分自己的权限，或者直接或间接地降低他提升的管理员的权限，则为真
			CanChangeInfo:       j.BotAdministratorRights.CanChangeInfo,       // 如果用户被允许更改聊天标题、照片和其他设置，则为真
			CanInviteUsers:      j.BotAdministratorRights.CanInviteUsers,      // 如果用户被允许邀请新用户加入聊天，则为真
			CanPostMessages:     j.BotAdministratorRights.CanPostMessages,     // 可选。如果管理员可以在频道中发帖，则为真；仅适用于频道
			CanEditMessages:     j.BotAdministratorRights.CanEditMessages,     // 可选。如果管理员可以编辑其他用户的消息并可以固定消息，则为真；仅适用于频道
			CanPinMessages:      j.BotAdministratorRights.CanPinMessages,      // 可选。如果用户被允许固定消息，则为真；仅适用于群组和超级群组
			CanPostStories:      j.BotAdministratorRights.CanPostStories,      // 可选。如果管理员可以在频道中发布故事，则为真；仅适用于频道
			CanEditStories:      j.BotAdministratorRights.CanEditStories,      // 可选。如果管理员可以编辑其他用户发布的故事，则为真；仅适用于频道
			CanDeleteStories:    j.BotAdministratorRights.CanDeleteStories,    // 可选。如果管理员可以删除其他用户发布的故事，则为真；仅适用于频道
			CanManageTopics:     j.BotAdministratorRights.CanManageTopics,     // 可选。如果用户被允许创建、重命名、关闭和重新打开论坛主题，则为真；仅适用于超级群组
		}, // 可选。一个 JSON 序列化的对象，列出机器人在聊天中所需的管理员权限。权限必须是 user_administrator_rights 的子集。如果未指定，则不应用任何额外限制。
		BotIsMember:     j.BotIsMember,     // 可选。传递 true 以请求与机器人作为成员的聊天。否则，不应用任何额外限制。
		RequestTitle:    j.RequestTitle,    // 可选。传递 true 以请求聊天标题。
		RequestUsername: j.RequestUsername, // 可选。传递 true 以请求聊天的用户名。
		RequestPhoto:    j.RequestPhoto,    // 可选。传递 true 以请求聊天的照片。
	}

}
