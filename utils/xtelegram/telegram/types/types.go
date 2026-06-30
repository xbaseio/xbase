package types

import (
	"encoding/json"
)

// APIResponse 是来自 Telegram API 的响应，结果以原始格式存储。
type APIResponse struct {
	Ok          bool                `json:"ok"`                    // 请求是否成功。
	Result      json.RawMessage     `json:"result,omitempty"`      // 可选字段，包含响应的原始结果。
	ErrorCode   int                 `json:"error_code,omitempty"`  // 可选字段，错误代码（如果请求失败）。
	Description string              `json:"description,omitempty"` // 可选字段，错误描述（如果请求失败）。
	Parameters  *ResponseParameters `json:"parameters,omitempty"`  // 可选字段，响应的额外参数。
}

// Update 表示从 GetUpdates 获取的更新响应。
type Update struct {
	UpdateID                int                          `json:"update_id"`                           // 更新的唯一标识符。从某个正数开始递增。使用 Webhooks 时可用该 ID 忽略重复更新或恢复正确的更新顺序。如果至少一周没有新更新，则下一个更新的 ID 将随机选择而非顺序递增。
	Message                 *Message                     `json:"message,omitempty"`                   // 可选字段。新收到的消息，可为任何类型（文本、照片、贴纸等）。
	EditedMessage           *Message                     `json:"edited_message,omitempty"`            // 可选字段。已知的消息被编辑后的新版本。
	ChannelPost             *Message                     `json:"channel_post,omitempty"`              // 可选字段。新收到的频道消息，可为任何类型（文本、照片、贴纸等）。
	EditedChannelPost       *Message                     `json:"edited_channel_post,omitempty"`       // 可选字段。已知频道消息被编辑后的新版本。
	BusinessConnection      *BusinessConnection          `json:"business_connection,omitempty"`       // 可选字段。表示机器人与业务账户的连接或断开，或者用户编辑了与机器人的现有连接。
	BusinessMessage         *Message                     `json:"business_message,omitempty"`          // 可选字段。来自连接的业务账户的新非服务消息。
	EditedBusinessMessage   *Message                     `json:"edited_business_message,omitempty"`   // 可选字段。来自连接业务账户的消息被编辑后的新版本。
	DeletedBusinessMessages *BusinessMessagesDeleted     `json:"deleted_business_messages,omitempty"` // 可选字段。来自连接业务账户的消息被删除。
	MessageReaction         *MessageReactionUpdated      `json:"message_reaction,omitempty"`          // 可选字段。用户更改了消息的反应。机器人必须是聊天的管理员，并在 allowed_updates 列表中显式指定 "message_reaction" 才能接收这些更新。机器人设置的反应不会触发更新。
	MessageReactionCount    *MessageReactionCountUpdated `json:"message_reaction_count,omitempty"`    // 可选字段。带匿名反应的消息反应发生了变化。机器人必须是聊天的管理员，并在 allowed_updates 列表中显式指定 "message_reaction_count" 才能接收这些更新。
	InlineQuery             *InlineQuery                 `json:"inline_query,omitempty"`              // 可选字段。新收到的 Inline 查询。
	ChosenInlineResult      *ChosenInlineResult          `json:"chosen_inline_result,omitempty"`      // 可选字段。用户选择的 Inline 查询结果并发送给聊天伙伴。请参阅文档了解如何启用这些更新。
	CallbackQuery           *CallbackQuery               `json:"callback_query,omitempty"`            // 可选字段。新收到的回调查询。
	ShippingQuery           *ShippingQuery               `json:"shipping_query,omitempty"`            // 可选字段。新收到的运费查询，仅适用于具有灵活价格的发票。
	PreCheckoutQuery        *PreCheckoutQuery            `json:"pre_checkout_query,omitempty"`        // 可选字段。新收到的结账前查询，包含有关结账的完整信息。
	Poll                    *Poll                        `json:"poll,omitempty"`                      // 可选字段。新的投票状态。机器人仅接收已停止的投票和机器人自己发送的投票的更新。
	PollAnswer              *PollAnswer                  `json:"poll_answer,omitempty"`               // 可选字段。用户在非匿名投票中更改了答案。机器人仅接收自己发送的投票的投票更新。
	MyChatMember            *ChatMemberUpdated           `json:"my_chat_member,omitempty"`            // 可选字段。机器人的聊天成员状态在某个聊天中更新。对于私人聊天，仅在用户阻止或取消阻止机器人时接收该更新。
	ChatMember              *ChatMemberUpdated           `json:"chat_member,omitempty"`               // 可选字段。聊天成员的状态在某个聊天中更新。机器人必须是聊天的管理员，并在 allowed_updates 列表中显式指定 "chat_member" 才能接收这些更新。
	ChatJoinRequest         *ChatJoinRequest             `json:"chat_join_request,omitempty"`         // 可选字段。有人请求加入聊天。机器人必须拥有 can_invite_users 管理员权限才能接收这些更新。
	ChatBoost               *ChatBoostUpdated            `json:"chat_boost,omitempty"`                // 可选字段。聊天的加速状态被添加或更改。机器人必须是聊天的管理员才能接收这些更新。
	RemovedChatBoost        *ChatBoostRemoved            `json:"removed_chat_boost,omitempty"`        // 可选字段。某个聊天的加速状态被移除。机器人必须是聊天的管理员才能接收这些更新。
}

// WebhookInfo 描述 Webhook 的当前状态。
type WebhookInfo struct {
	URL                          string   `json:"url"`                                       // Webhook 的 URL。如果未设置 Webhook，则可能为空。
	HasCustomCertificate         bool     `json:"has_custom_certificate"`                    // 如果为 Webhook 提供了自定义证书，则为 true。
	PendingUpdateCount           int      `json:"pending_update_count"`                      // 等待交付的更新数量。
	IPAddress                    string   `json:"ip_address,omitempty"`                      // 可选字段。当前使用的 Webhook 的 IP 地址。
	LastErrorDate                int      `json:"last_error_date,omitempty"`                 // 可选字段。最近一次尝试通过 Webhook 交付更新时发生错误的 Unix 时间。
	LastErrorMessage             string   `json:"last_error_message,omitempty"`              // 可选字段。最近一次尝试通过 Webhook 交付更新时的错误信息。
	LastSynchronizationErrorDate int      `json:"last_synchronization_error_date,omitempty"` // 可选字段。最近一次尝试与 Telegram 数据中心同步更新时发生错误的 Unix 时间。
	MaxConnections               int      `json:"max_connections,omitempty"`                 // 可选字段。允许同时建立的最大 HTTPS 连接数以推送更新。
	AllowedUpdates               []string `json:"allowed_updates,omitempty"`                 // 可选字段。机器人订阅的更新类型列表。默认为所有更新类型，除了 chat_member。
}

// User 表示一个 Telegram 用户或机器人。
type User struct {
	ID                      int64  `json:"id"`                                    // 用户或机器人的唯一标识符。此数字可能超过 32 位，因此某些编程语言可能会在处理时出现问题/静默错误。但最多只有 52 位有效位，因此 64 位整数或双精度浮点类型可以安全存储此标识符。
	IDString                string `json:"-"`                                     // 字符串形式的 ID。
	IsBot                   bool   `json:"is_bot"`                                // 如果此用户是机器人，则为 true。
	FirstName               string `json:"first_name"`                            // 用户或机器人的名字。
	LastName                string `json:"last_name,omitempty"`                   // 可选字段。用户或机器人的姓氏。
	UserName                string `json:"username,omitempty"`                    // 可选字段。用户或机器人的用户名。
	LanguageCode            string `json:"language_code,omitempty"`               // 可选字段。用户的语言代码（IETF 语言标签）(https://en.wikipedia.org/wiki/IETF_language_tag)。
	IsPremium               bool   `json:"is_premium,omitempty"`                  // 可选字段。如果用户是 Telegram 高级用户，则为 true。
	AddedToAttachmentMenu   bool   `json:"added_to_attachment_menu,omitempty"`    // 可选字段。如果用户将机器人添加到附件菜单，则为 true。
	CanJoinGroups           bool   `json:"can_join_groups,omitempty"`             // 可选字段。如果机器人可以被邀请加入群组，则为 true。仅在 getMe 请求中返回。
	CanReadAllGroupMessages bool   `json:"can_read_all_group_messages,omitempty"` // 可选字段。如果机器人禁用了隐私模式，则为 true。仅在 getMe 请求中返回。
	SupportsInlineQueries   bool   `json:"supports_inline_queries,omitempty"`     // 可选字段。如果机器人支持 Inline 查询，则为 true。仅在 getMe 请求中返回。
	CanConnectToBusiness    bool   `json:"can_connect_to_business,omitempty"`     // 可选字段。如果机器人可以连接到 Telegram 业务账户以接收其消息，则为 true。仅在 getMe 请求中返回。
}

// Chat 表示一个聊天。
type Chat struct {
	ID        int64  `json:"id"`                   // 此聊天的唯一标识符。该数字可能超过 32 位，对于某些编程语言可能会有难以处理或静默缺陷的问题。但它最多只有 52 位有效位，因此可以安全地使用有符号的 64 位整数或双精度浮点类型来存储此标识符。
	IDString  string `json:"-"`                    // 字符串形式的 ID。
	Type      string `json:"type"`                 // 聊天的类型，可以是 “private”（私聊）、“group”（普通群组）、“supergroup”（超级群组）或 “channel”（频道）。
	Title     string `json:"title,omitempty"`      // 可选。标题，适用于超级群组、频道和群组聊天。
	UserName  string `json:"username,omitempty"`   // 可选。用户名，适用于私聊、超级群组和频道（如果可用）。
	FirstName string `json:"first_name,omitempty"` // 可选。在私人聊天中，另一方的名字。
	LastName  string `json:"last_name,omitempty"`  // 可选。在私人聊天中，另一方的姓氏。
	IsForum   bool   `json:"is_forum,omitempty"`   // 可选。如果超级群组启用了主题（论坛模式），则为 true。
}

// ChatFullInfo 包含关于聊天的完整信息。
type ChatFullInfo struct {
	ID                                 int64                 `json:"id"`                                                // 此聊天的唯一标识符。该数字可能超过 32 位，对于某些编程语言可能会有难以处理或静默缺陷的问题。但它最多只有 52 位有效位，因此可以安全地使用有符号的 64 位整数或双精度浮点类型来存储此标识符。
	IDString                           string                `json:"-"`                                                 // 字符串形式的 ID。
	Type                               string                `json:"type"`                                              // 聊天的类型，可以是 “private”（私聊）、“group”（普通群组）、“supergroup”（超级群组）或 “channel”（频道）。
	Title                              string                `json:"title,omitempty"`                                   // 可选。标题，适用于超级群组、频道和群组聊天。
	UserName                           string                `json:"username,omitempty"`                                // 可选。用户名，适用于私聊、超级群组和频道（如果可用）。
	FirstName                          string                `json:"first_name,omitempty"`                              // 可选。在私人聊天中，另一方的名字。
	LastName                           string                `json:"last_name,omitempty"`                               // 可选。在私人聊天中，另一方的姓氏。
	IsForum                            bool                  `json:"is_forum,omitempty"`                                // 可选。如果超级群组启用了主题（论坛模式），则为 true。
	Photo                              *ChatPhoto            `json:"photo,omitempty"`                                   // 可选。聊天照片。仅在调用 getChat 时返回。
	ActiveUsernames                    []string              `json:"active_usernames,omitempty"`                        // 可选。如果非空，则是所有活动聊天用户名的列表；适用于私聊、超级群组和频道。仅在调用 getChat 时返回。
	Birthdate                          *Birthdate            `json:"birthdate,omitempty"`                               // 可选。对于私人聊天，用户的出生日期。仅在调用 getChat 时返回。
	BusinessIntro                      *BusinessIntro        `json:"business_intro,omitempty"`                          // 可选。对于与商业账户的私聊，商业简介。仅在调用 getChat 时返回。
	BusinessLocation                   *BusinessLocation     `json:"business_location,omitempty"`                       // 可选。对于与商业账户的私聊，商业位置。仅在调用 getChat 时返回。
	BusinessOpeningHours               *BusinessOpeningHours `json:"business_opening_hours,omitempty"`                  // 可选。对于与商业账户的私聊，商业营业时间。仅在调用 getChat 时返回。
	PersonalChat                       *Chat                 `json:"personal_chat,omitempty"`                           // 可选。对于私人聊天，用户的私人频道。仅在调用 getChat 时返回。
	AvailableReactions                 []ReactionType        `json:"available_reactions,omitempty"`                     // 可选。聊天中允许的可用反应列表。如果省略，则允许所有表情符号反应。仅在调用 getChat 时返回。
	AccentColorId                      int                   `json:"accent_color_id,omitempty"`                         // 聊天名称、聊天照片背景、回复标题和链接预览的强调颜色标识符。请参阅强调颜色的详细信息。仅在调用 getChat 时返回。
	MaxReactionCount                   int                   `json:"max_reaction_count,omitempty"`                      // 聊天中消息上可以设置的最大反应数。
	BackgroundCustomEmojiId            string                `json:"background_custom_emoji_id,omitempty"`              // 可选。聊天为回复标题和链接预览背景选择的自定义表情符号标识符。仅在调用 getChat 时返回。
	ProfileAccentColorId               int                   `json:"profile_accent_color_id,omitempty"`                 // 可选。聊天的个人资料背景的强调颜色标识符。请参阅个人资料强调颜色的详细信息。仅在调用 getChat 时返回。
	ProfileBackgroundCustomEmojiId     string                `json:"profile_background_custom_emoji_id,omitempty"`      // 可选。聊天为其个人资料背景选择的自定义表情符号的标识符。仅在调用 getChat 时返回。
	EmojiStatusCustomEmojiId           string                `json:"emoji_status_custom_emoji_id,omitempty"`            // 可选。聊天或私人聊天中另一方的表情状态的自定义表情符号标识符。仅在调用 getChat 时返回。
	EmojiStatusExpirationDate          int64                 `json:"emoji_status_expiration_date,omitempty"`            // 可选。在 Unix 时间中，聊天或私人聊天中另一方的表情状态的到期日期（如果有）。仅在调用 getChat 时返回。
	Bio                                string                `json:"bio,omitempty"`                                     // 可选。在私人聊天中，另一方的个人简介。仅在调用 getChat 时返回。
	HasPrivateForwards                 bool                  `json:"has_private_forwards,omitempty"`                    // 可选。如果为 true，则私人聊天中另一方的隐私设置仅允许在与用户的聊天中使用 tg://user?id=<user_id> 链接。仅在调用 getChat 时返回。
	HasRestrictedVoiceAndVideoMessages bool                  `json:"has_restricted_voice_and_video_messages,omitempty"` // 可选。如果为 true，则私人聊天中另一方的隐私设置限制发送语音和视频消息。仅在调用 getChat 时返回。
	JoinToSendMessages                 bool                  `json:"join_to_send_messages,omitempty"`                   // 可选。如果为 true，则用户需要加入超级群组才能发送消息。仅在调用 getChat 时返回。
	JoinByRequest                      bool                  `json:"join_by_request,omitempty"`                         // 可选。如果为 true，则所有直接加入超级群组的用户都需要由超级群组管理员批准。仅在调用 getChat 时返回。
	Description                        string                `json:"description,omitempty"`                             // 可选。描述，适用于群组、超级群组和频道聊天。仅在调用 getChat 时返回。
	InviteLink                         string                `json:"invite_link,omitempty"`                             // 可选。主邀请链接，适用于群组、超级群组和频道聊天。仅在调用 getChat 时返回。
	PinnedMessage                      *Message              `json:"pinned_message,omitempty"`                          // 可选。最近置顶的消息（按发送日期）。仅在调用 getChat 时返回。
	Permissions                        *ChatPermissions      `json:"permissions,omitempty"`                             // 可选。默认聊天成员权限，适用于群组和超级群组。仅在调用 getChat 时返回。
	SlowModeDelay                      int                   `json:"slow_mode_delay,omitempty"`                         // 可选。对于超级群组，未授权用户连续发送消息的最小延迟（以秒为单位）。仅在调用 getChat 时返回。
	UnRestrictBoostCount               int                   `json:"unrestrict_boost_count,omitempty"`                  // 可选。对于超级群组，非管理员用户需要添加的最小加速次数，以忽略慢速模式和聊天权限。仅在调用 getChat 时返回。
	MessageAutoDeleteTime              int                   `json:"message_auto_delete_time,omitempty"`                // 可选。聊天中所有消息将在此时间后自动删除（以秒为单位）。仅在调用 getChat 时返回。
	HasAggressiveAntiSpamEnabled       bool                  `json:"has_aggressive_anti_spam_enabled,omitempty"`        // 可选。如果为 true，则在超级群组中启用了积极的反垃圾检查。此字段仅对聊天管理员可见。仅在调用 getChat 时返回。
	HasHiddenMembers                   bool                  `json:"has_hidden_members,omitempty"`                      // 可选。如果为 true，则非管理员只能获取聊天中的机器人和管理员列表。仅在调用 getChat 时返回。
	HasProtectedContent                bool                  `json:"has_protected_content,omitempty"`                   // 可选。如果为 true，则无法将来自聊天的消息转发到其他聊天。仅在调用 getChat 时返回。
	HasVisibleHistory                  bool                  `json:"has_visible_history,omitempty"`                     // 可选。如果为 true，则新聊天成员将可以访问旧消息；仅对聊天管理员可见。仅在调用 getChat 时返回。
	StickerSetName                     string                `json:"sticker_set_name,omitempty"`                        // 可选。对于超级群组，群组贴纸集的名称。仅在调用 getChat 时返回。
	CanSetStickerSet                   bool                  `json:"can_set_sticker_set,omitempty"`                     // 可选。如果机器人可以更改群组贴纸集，则为 true。仅在调用 getChat 时返回。
	CustomEmojiStickerSetName          string                `json:"custom_emoji_sticker_set_name,omitempty"`           // 可选。对于超级群组，群组的自定义表情符号贴纸集的名称。此贴纸集中的自定义表情符号可以由群组中的所有用户和机器人使用。仅在调用 getChat 时返回。
	LinkedChatID                       int64                 `json:"linked_chat_id,omitempty"`                          // 可选。链接聊天的唯一标识符，例如频道的讨论组标识符，反之亦然，适用于超级群组和频道聊天。此标识符可能大于 32 位，对于某些编程语言可能会有难以处理或静默缺陷的问题。但它小于 52 位，因此可以安全地使用有符号的 64 位整数或双精度浮点类型来存储此标识符。仅在调用 getChat 时返回。
	Location                           *ChatLocation         `json:"location,omitempty"`                                // 可选。对于超级群组，与超级群组连接的位置。仅在调用 getChat 时返回。
}

// Message 表示一条消息。
type Message struct {
	MessageID                     int                            `json:"message_id"`                                  // 聊天中此消息的唯一标识符。
	MessageThreadID               int                            `json:"message_thread_id,omitempty"`                 // 可选。消息所属的消息线程的唯一标识符；仅适用于超级群组。
	From                          *User                          `json:"from,omitempty"`                              // 可选。消息的发送者；对于发送到频道的消息为空。为了向后兼容，如果消息是以聊天名义发送的，则在非频道聊天中该字段包含一个虚拟发送者用户。
	SenderChat                    *Chat                          `json:"sender_chat,omitempty"`                       // 可选。以聊天名义发送的消息的发送者。例如，频道本身对于频道的消息、匿名群组管理员所在的超级群组本身、自动转发到讨论组的链接频道。为了向后兼容，如果消息是以聊天名义发送的，则在非频道聊天中该字段包含一个虚拟发送者用户。
	SenderBoostCount              int                            `json:"sender_boost_count,omitempty"`                // 可选。如果消息的发送者为聊天加速了，则为用户添加的加速次数。
	SenderBusinessBot             *User                          `json:"sender_business_bot,omitempty"`               // 可选。代表商业账户实际发送消息的机器人。仅对以连接的商业账户名义发送的传出消息可用。
	Date                          int64                          `json:"date"`                                        // 消息发送的日期（Unix 时间）。
	BusinessConnectionId          string                         `json:"business_connection_id,omitempty"`            // 可选。消息接收来源的商业连接的唯一标识符。如果非空，则消息属于对应商业账户的聊天，与可能共享相同标识符的任何潜在机器人聊天独立。
	Chat                          Chat                           `json:"chat"`                                        // 消息所属的会话。
	ForwardOrigin                 *MessageOrigin                 `json:"forward_origin,omitempty"`                    // 可选。转发消息的原始消息的信息。
	IsTopicMessage                bool                           `json:"is_topic_message,omitempty"`                  // 可选。如果消息是发送到论坛主题的，则为 true。
	IsAutomaticForward            bool                           `json:"is_automatic_forward,omitempty"`              // 可选。如果消息是自动转发到连接的讨论组的频道帖子，则为 true。
	ReplyToMessage                *Message                       `json:"reply_to_message,omitempty"`                  // 可选。对于回复消息，原始消息。注意，即使该消息本身是回复，reply_to_message 字段中的 Message 对象也不会包含进一步的 reply_to_message 字段。
	ExternalReply                 *ExternalReplyInfo             `json:"external_reply,omitempty"`                    // 可选。被回复的消息的信息，可能来自另一个聊天或论坛主题。
	Quote                         *TextQuote                     `json:"quote,omitempty"`                             // 可选。对于引用原始消息部分内容的回复，消息中引用的部分。
	ReplyToStory                  *Story                         `json:"reply_to_story,omitempty"`                    // 可选。对于回复的故事，原始故事。
	ViaBot                        *User                          `json:"via_bot,omitempty"`                           // 可选。通过该机器人发送消息的用户。
	EditDate                      int64                          `json:"edit_date,omitempty"`                         // 可选。消息最后编辑的日期（Unix 时间）。
	HasProtectedContent           bool                           `json:"has_protected_content,omitempty"`             // 可选。如果消息不能被转发，则为 true。
	IsFromOffline                 bool                           `json:"is_from_offline,omitempty"`                   // 可选。如果消息由隐式操作发送，例如作为离线消息、招呼消息或计划消息，则为 true。
	MediaGroupID                  string                         `json:"media_group_id,omitempty"`                    // 可选。此消息所属的媒体消息组的唯一标识符。
	AuthorSignature               string                         `json:"author_signature,omitempty"`                  // 可选。频道消息的作者签名，或匿名群组管理员的自定义标题。
	Text                          string                         `json:"text,omitempty"`                              // 可选。对于文本消息，消息的实际 UTF-8 文本。
	Entities                      []*MessageEntity               `json:"entities,omitempty"`                          // 可选。对于文本消息，出现在文本中的特殊实体（如用户名、URL、机器人命令等）。
	LinkPreviewOptions            *LinkPreviewOptions            `json:"link_preview_options,omitempty"`              // 可选。用于消息链接预览生成的选项，如果是文本消息并且更改了链接预览选项。
	EffectId                      string                         `json:"effect_id,omitempty"`                         // 可选。添加到消息的消息效果的唯一标识符。
	Animation                     *Animation                     `json:"animation,omitempty"`                         // 可选。消息是一段动画，包含有关动画的信息。为了向后兼容，当此字段设置时，document 字段也会被设置。
	Audio                         *Audio                         `json:"audio,omitempty"`                             // 可选。消息是音频文件，包含关于该文件的信息。
	Document                      *Document                      `json:"document,omitempty"`                          // 可选。消息是一般文件，包含关于该文件的信息。
	Photo                         []*PhotoSize                   `json:"photo,omitempty"`                             // 可选。消息是一张照片，包含照片的可用大小。
	Sticker                       *Sticker                       `json:"sticker,omitempty"`                           // 可选。消息是一张贴纸，包含关于该贴纸的信息。
	Story                         *Story                         `json:"story,omitempty"`                             // 可选。消息是转发的故事。
	Video                         *Video                         `json:"video,omitempty"`                             // 可选。消息是视频，包含关于该视频的信息。
	VideoNote                     *VideoNote                     `json:"video_note,omitempty"`                        // 可选。消息是视频便笺，包含关于视频消息的信息。
	Voice                         *Voice                         `json:"voice,omitempty"`                             // 可选。消息是语音消息，包含关于该文件的信息。
	Caption                       string                         `json:"caption,omitempty"`                           // 可选。动画、音频、文件、照片、视频或语音的说明。
	CaptionEntities               []*MessageEntity               `json:"caption_entities,omitempty"`                  // 可选。对于有说明的消息，说明中出现的特殊实体（如用户名、URL、机器人命令等）。
	ShowCaptionAboveMedia         bool                           `json:"show_caption_above_media,omitempty"`          // 可选。如果说明必须显示在消息媒体上方，则为 true。
	HasMediaSpoiler               bool                           `json:"has_media_spoiler,omitempty"`                 // 可选。如果消息媒体被剧透动画覆盖，则为 true。
	Contact                       *Contact                       `json:"contact,omitempty"`                           // 可选。消息是共享的联系人，包含关于该联系人的信息。
	Dice                          *Dice                          `json:"dice,omitempty"`                              // 可选。消息是一个带有随机值的骰子。
	Game                          *Game                          `json:"game,omitempty"`                              // 可选。消息是一个游戏，包含关于该游戏的信息。
	Poll                          *Poll                          `json:"poll,omitempty"`                              // 可选。消息是一个原生投票，包含有关该投票的信息。
	Venue                         *Venue                         `json:"venue,omitempty"`                             // 可选。消息是一个地点，包含关于该地点的信息。为了向后兼容，当此字段设置时，location 字段也会被设置。
	Location                      *Location                      `json:"location,omitempty"`                          // 可选。消息是共享的位置，包含关于该位置的信息。
	NewChatMembers                []*User                        `json:"new_chat_members,omitempty"`                  // 可选。被添加到群组或超级群组的新成员及其信息（机器人本身可能是这些成员之一）。
	LeftChatMember                *User                          `json:"left_chat_member,omitempty"`                  // 可选。一名成员被移出群组的信息（该成员可能是机器人本身）。
	NewChatTitle                  string                         `json:"new_chat_title,omitempty"`                    // 可选。聊天标题被更改为此值。
	NewChatPhoto                  []*PhotoSize                   `json:"new_chat_photo,omitempty"`                    // 可选。聊天照片已更改为此值
	DeleteChatPhoto               bool                           `json:"delete_chat_photo,omitempty"`                 // 可选。服务消息：聊天照片已删除
	GroupChatCreated              bool                           `json:"group_chat_created,omitempty"`                // 可选。服务消息：群组已创建
	SuperGroupChatCreated         bool                           `json:"supergroup_chat_created,omitempty"`           // 可选。服务消息：超群已创建。此字段不能在通过更新收到的消息中接收，因为机器人在创建时不能成为超群的成员。它只能在 reply_to_message 中找到，如果有人回复直接创建的超群中的第一条消息。
	ChannelChatCreated            bool                           `json:"channel_chat_created,omitempty"`              // 可选。服务消息：频道已创建。此字段不能在通过更新收到的消息中接收，因为机器人在创建时不能成为频道的成员。它只能在 reply_to_message 中找到，如果有人回复频道中的第一条消息。
	MessageAutoDeleteTimerChanged *MessageAutoDeleteTimerChanged `json:"message_auto_delete_timer_changed,omitempty"` // 可选。服务消息：聊天中的自动删除计时器设置已更改
	MigrateToChatID               int64                          `json:"migrate_to_chat_id,omitempty"`                // 可选。该群组已迁移到具有指定标识符的超群。该数字可能具有超过 32 个有效位的长度，某些编程语言可能会在解释时遇到问题。但它最多具有 52 个有效位，因此使用带符号的 64 位整数或双精度浮点数类型来存储此标识符是安全的。
	MigrateFromChatID             int64                          `json:"migrate_from_chat_id,omitempty"`              // 可选。该超群已从具有指定标识符的群组迁移。该数字可能具有超过 32 个有效位的长度，某些编程语言可能会在解释时遇到问题。但它最多具有 52 个有效位，因此使用带符号的 64 位整数或双精度浮点数类型来存储此标识符是安全的。
	PinnedMessage                 *Message                       `json:"pinned_message,omitempty"`                    // 可选。指定的消息已被置顶。请注意，此字段中的消息对象不会包含进一步的 reply_to_message 字段，即使它本身是回复。
	Invoice                       *Invoice                       `json:"invoice,omitempty"`                           // 可选。消息是付款的发票，关于发票的信息。
	SuccessfulPayment             *SuccessfulPayment             `json:"successful_payment,omitempty"`                // 可选。消息是关于成功付款的服务消息，关于付款的信息。
	UsersShared                   *UsersShared                   `json:"users_shared,omitempty"`                      // 可选。服务消息：与机器人共享了一个用户
	ChatShared                    *ChatShared                    `json:"chat_shared,omitempty"`                       // 可选。服务消息：与机器人共享了一个聊天
	ConnectedWebsite              string                         `json:"connected_website,omitempty"`                 // 可选。用户已登录的网站的域名。
	WriteAccessAllowed            *WriteAccessAllowed            `json:"write_access_allowed,omitempty"`              // 可选。服务消息：用户允许添加到附件菜单的机器人写消息
	PassportData                  *PassportData                  `json:"passport_data,omitempty"`                     // 可选。Telegram 护照数据
	ProximityAlertTriggered       *ProximityAlertTriggered       `json:"proximity_alert_triggered,omitempty"`         // 可选。服务消息。聊天中的用户在共享实时位置时触发了另一个用户的接近警报。
	BoostAdded                    *ChatBoostAdded                `json:"boost_added,omitempty"`                       // 可选。服务消息：用户提升了聊天
	ChatBackgroundSet             *ChatBackground                `json:"chat_background_set,omitempty"`               // 可选。服务消息：聊天背景已设置
	ForumTopicCreated             *ForumTopicCreated             `json:"forum_topic_created,omitempty"`               // 可选。服务消息：论坛话题已创建
	ForumTopicEdited              *ForumTopicEdited              `json:"forum_topic_edited,omitempty"`                // 可选。服务消息：论坛话题已编辑
	ForumTopicClosed              *ForumTopicClosed              `json:"forum_topic_closed,omitempty"`                // 可选。服务消息：论坛话题已关闭
	ForumTopicReopened            *ForumTopicReopened            `json:"forum_topic_reopened,omitempty"`              // 可选。服务消息：论坛话题已重新开放
	GeneralForumTopicHidden       *GeneralForumTopicHidden       `json:"general_forum_topic_hidden,omitempty"`        // 可选。服务消息：'常规' 论坛话题已隐藏
	GeneralForumTopicUnhidden     *GeneralForumTopicUnhidden     `json:"general_forum_topic_unhidden,omitempty"`      // 可选。服务消息：'常规' 论坛话题已取消隐藏
	GiveawayCreated               *GiveawayCreated               `json:"giveaway_created,omitempty"`                  // 可选。服务消息：已创建计划赠品
	Giveaway                      *Giveaway                      `json:"giveaway,omitempty"`                          // 可选。消息是计划赠品消息
	GiveawayWinners               *GiveawayWinners               `json:"giveaway_winners,omitempty"`                  // 可选。已完成的公开获奖赠品
	GiveawayCompleted             *GiveawayCompleted             `json:"giveaway_completed,omitempty"`                // 可选。没有公开获奖的赠品已完成
	VideoChatScheduled            *VideoChatScheduled            `json:"video_chat_scheduled,omitempty"`              // 可选。服务消息：视频聊天已安排
	VideoChatStarted              *VideoChatStarted              `json:"video_chat_started,omitempty"`                // 可选。服务消息：视频聊天已开始
	VideoChatEnded                *VideoChatEnded                `json:"video_chat_ended,omitempty"`                  // 可选。服务消息：视频聊天已结束
	VideoChatParticipantsInvited  *VideoChatParticipantsInvited  `json:"video_chat_participants_invited,omitempty"`   // 可选。服务消息：新参与者已被邀请加入视频聊天
	WebAppData                    *WebAppData                    `json:"web_app_data,omitempty"`                      // 可选。服务消息：Web 应用发送的数据
	ReplyMarkup                   *InlineKeyboardMarkup          `json:"reply_markup,omitempty"`                      // 可选。附加到消息的内联键盘。Login_url 按钮表示为普通 URL 按钮。
}

// MessageID 表示唯一的消息标识符。
type MessageID struct {
	MessageID int `json:"message_id"` // 唯一的消息标识符
}

// InaccessibleMessage 描述了一条已被删除或对机器人不可访问的消息。
type InaccessibleMessage struct {
	Chat      Chat  `json:"chat"`       // 消息所属的聊天
	MessageID int   `json:"message_id"` // 聊天中唯一的消息标识符
	Date      int64 `json:"date"`       // 始终为 0。该字段可用于区分常规消息和不可访问消息。
}

// MaybeInaccessibleMessage 描述了一条可能对机器人不可访问的消息。它可以是其中之一
type MaybeInaccessibleMessage struct {
	Message
	InaccessibleMessage
}

// MessageEntity 表示文本消息中的一个特殊实体。例如，标签、用户名、URL 等
type MessageEntity struct {
	Type          string `json:"type"`                      // 实体的类型。目前可以是“mention”（@用户名）、“hashtag”（#标签）、“hashtag”（$USD）、“bot_command”（/start@jobs_bot）、“url”（https://telegram.org）、“email”（do-not-reply@telegram.org）、“phone_number”（+1-212-555-0123）、“bold”（粗体文本）、“italic”（斜体文本）、“underline”（下划线文本）、“strikethrough”（删除线文本）、“spoiler”（剧透消息）、“code”（等宽字符串）、“pre”（等宽块）、“text_link”（可点击文本 URL）、“text_mention”（对没有用户名的用户）
	Offset        int    `json:"offset"`                    // 实体开始位置的 UTF-16 代码单元偏移量
	Length        int    `json:"length"`                    // 实体的长度，以 UTF-16 代码单元为单位
	URL           string `json:"url,omitempty"`             // 可选。仅适用于“text_link”，用户点击文本后将打开的 URL
	User          *User  `json:"user,omitempty"`            // 可选。仅适用于“text_mention”，被提及的用户
	Language      string `json:"language,omitempty"`        // 可选。仅适用于“pre”，实体文本的编程语言
	CustomEmojiId string `json:"custom_emoji_id,omitempty"` // 可选。仅适用于“custom_emoji”，自定义表情的唯一标识符。使用 getCustomEmojiStickers 获取关于表情的完整信息
}

// TextQuote 包含有关被引用消息的部分信息，该消息被给定消息回复。
type TextQuote struct {
	Text     string          `json:"text"`                // 被引用消息的文本
	Entities []MessageEntity `json:"entities,omitempty"`  // 可选。引用中出现的特殊实体。目前，仅保留粗体、斜体、下划线、删除线、剧透和自定义表情实体。
	Position int             `json:"position"`            // 按发送者指定的 UTF-16 代码单元大致引用位置
	IsManual bool            `json:"is_manual,omitempty"` // 可选。如果引用是由消息发送者手动选择的，则为真。否则，引用是由服务器自动添加的。
}

// ExternalReplyInfo 包含有关正在回复的消息的信息，该消息可能来自其他聊天或论坛主题。
type ExternalReplyInfo struct {
	Origin             MessageOrigin       `json:"origin"`                         // 被回复消息的来源
	Chat               *Chat               `json:"chat,omitempty"`                 // 可选。原始消息所属的聊天。仅在聊天为超群或频道时可用。
	MessageID          int                 `json:"message_id,omitempty"`           // 可选。原始聊天中的唯一消息标识符。仅在原始聊天为超群或频道时可用。
	LinkPreviewOptions *LinkPreviewOptions `json:"link_preview_options,omitempty"` // 可选。用于生成原始消息链接预览的选项，如果它是文本消息
	Animation          *Animation          `json:"animation,omitempty"`            // 可选。消息是动画，关于动画的信息
	Audio              *Audio              `json:"audio,omitempty"`                // 可选。消息是音频文件，关于该文件的信息
	Document           *Document           `json:"document,omitempty"`             // 可选。消息是通用文件，关于该文件的信息
	Photo              []PhotoSize         `json:"photo,omitempty"`                // 可选。消息是照片，照片的可用尺寸
	Sticker            *Sticker            `json:"sticker,omitempty"`              // 可选。消息是贴纸，关于该贴纸的信息
	Story              *Story              `json:"story,omitempty"`                // 可选。消息是转发的故事
	Video              *Video              `json:"video,omitempty"`                // 可选。消息是视频，关于该视频的信息
	VideoNote          *VideoNote          `json:"video_note,omitempty"`           // 可选。消息是视频便签，关于视频消息的信息
	Voice              *Voice              `json:"voice,omitempty"`                // 可选。消息是语音消息，关于该文件的信息
	HasMediaSpoiler    bool                `json:"has_media_spoiler,omitempty"`    // 可选。如果消息媒体被剧透动画遮盖，则为真
	Contact            *Contact            `json:"contact,omitempty"`              // 可选。消息是共享的联系人，关于该联系人的信息
	Dice               *Dice               `json:"dice,omitempty"`                 // 可选。消息是带有随机值的骰子
	Game               *Game               `json:"game,omitempty"`                 // 可选。消息是游戏，关于该游戏的信息。更多关于游戏的信息
	Giveaway           *Giveaway           `json:"giveaway,omitempty"`             // 可选。消息是计划中的赠品，关于该赠品的信息
	GiveawayWinners    *GiveawayWinners    `json:"giveaway_winners,omitempty"`     // 可选。已完成的公开获奖赠品
	Invoice            *Invoice            `json:"invoice,omitempty"`              // 可选。消息是付款的发票，关于该发票的信息。
	Location           *Location           `json:"location,omitempty"`             // 可选。消息是共享位置，关于该位置的信息
	Poll               *Poll               `json:"poll,omitempty"`                 // 可选。消息是本地投票，关于该投票的信息
	Venue              *Venue              `json:"venue,omitempty"`                // 可选。消息是场地，关于该场地的信息
}

// ReplyParameters 描述正在发送的消息的回复参数。
type ReplyParameters struct {
	MessageID                int             `json:"message_id"`                            // 当前聊天中将被回复的消息的标识符，或者在指定的聊天 chat_id 中的标识符
	ChatID                   string          `json:"chat_id,omitempty"`                     // 可选。如果要回复的消息来自不同的聊天，唯一聊天标识符或频道的用户名（格式为 @channelusername）
	AllowSendingWithoutReply bool            `json:"allow_sending_without_reply,omitempty"` // 可选。如果消息应在指定的被回复消息未找到时发送，则传递 True；仅可用于同一聊天和论坛主题中的回复。
	Quote                    string          `json:"quote,omitempty"`                       // 可选。被回复消息的引用部分；在解析实体后为 0-1024 个字符。引用必须是被回复消息的确切子字符串，包括粗体、斜体、下划线、删除线、剧透和自定义表情实体。如果在原始消息中找不到引用，消息将发送失败。
	QuoteParseMode           string          `json:"quote_parse_mode,omitempty"`            // 可选。解析引用中实体的模式。有关详细信息，请参见格式选项。
	QuoteEntities            []MessageEntity `json:"quote_entities,omitempty"`              // 可选。出现在引用中的特殊实体的 JSON 序列化列表。可以替代 quote_parse_mode 指定。
	QuotePosition            int             `json:"quote_position,omitempty"`              // 可选。原始消息中引用的 UTF-16 代码单元位置
}

// MessageOrigin 描述消息的来源。可以是以下之一
type MessageOrigin struct {
	MessageOriginUser
	MessageOriginHiddenUser
	MessageOriginChat
	MessageOriginChannel
}

// MessageOriginUser 最初由已知用户发送。
type MessageOriginUser struct {
	Type       string `json:"type"`        // 消息来源的类型，始终为“user”
	Date       int64  `json:"date"`        // 消息最初发送的日期（Unix 时间）
	SenderUser User   `json:"sender_user"` // 最初发送消息的用户
}

// MessageOriginHiddenUser 最初由未知用户发送。
type MessageOriginHiddenUser struct {
	Type           string `json:"type"`             // 消息来源的类型，始终为“hidden_user”
	Date           int64  `json:"date"`             // 消息最初发送的日期（Unix 时间）
	SenderUserName string `json:"sender_user_name"` // 最初发送消息的用户名称
}

// MessageOriginChat 最初代表聊天发送到群组聊天的消息。
type MessageOriginChat struct {
	Type            string `json:"type"`                       // 消息来源的类型，始终为“chat”
	Date            int64  `json:"date"`                       // 消息最初发送的日期（Unix 时间）
	SenderChat      Chat   `json:"sender_chat"`                // 最初发送消息的聊天
	AuthorSignature string `json:"author_signature,omitempty"` // 可选。对于由匿名聊天管理员最初发送的消息，原始消息作者的签名
}

// MessageOriginChannel 最初发送到频道聊天的消息。
type MessageOriginChannel struct {
	Type            string `json:"type"`                       // 消息来源的类型，始终为“channel”
	Date            int64  `json:"date"`                       // 消息最初发送的日期（Unix 时间）
	SenderChat      Chat   `json:"sender_chat"`                // 消息最初发送的频道聊天
	MessageID       int    `json:"message_id"`                 // 聊天中唯一的消息标识符
	AuthorSignature string `json:"author_signature,omitempty"` // 可选。原始帖子的作者签名
}

// PhotoSize 表示照片或文件/贴纸缩略图的一种尺寸。
type PhotoSize struct {
	FileID       string `json:"file_id"`             // 该文件的标识符，可以用于下载或重用该文件
	FileUniqueID string `json:"file_unique_id"`      // 该文件的唯一标识符，应该在不同时间和不同机器人中保持不变。不能用于下载或重用该文件。
	Width        int    `json:"width"`               // 照片宽度
	Height       int    `json:"height"`              // 照片高度
	FileSize     int64  `json:"file_size,omitempty"` // 可选。文件大小（以字节为单位）
}

// Animation 表示动画文件（GIF 或 H.264/MPEG-4 AVC 视频，无声）。
type Animation struct {
	FileID       string     `json:"file_id"`             // 该文件的标识符，可以用于下载或重用该文件
	FileUniqueID string     `json:"file_unique_id"`      // 该文件的唯一标识符，应该在不同时间和不同机器人中保持不变。不能用于下载或重用该文件。
	Width        int        `json:"width"`               // 视频宽度，由发送者定义
	Height       int        `json:"height"`              // 视频高度，由发送者定义
	Duration     int        `json:"duration"`            // 视频持续时间（以秒为单位），由发送者定义
	Thumbnail    *PhotoSize `json:"thumb,omitempty"`     // 可选。发送者定义的动画缩略图
	FileName     string     `json:"file_name,omitempty"` // 可选。发送者定义的原始动画文件名
	MimeType     string     `json:"mime_type,omitempty"` // 可选。发送者定义的文件 MIME 类型
	FileSize     int64      `json:"file_size,omitempty"` // 可选。文件大小（以字节为单位）。可能大于 2^31，某些编程语言可能在解释时遇到困难/出现静默缺陷。但它最多具有 52 位有效位，因此使用带符号的 64 位整数或双精度浮点类型来存储此值是安全的。
}

// Audio 表示音频文件，Telegram 客户端将其视为音乐。
type Audio struct {
	FileID       string     `json:"file_id"`             // 该文件的标识符，可以用于下载或重用该文件
	FileUniqueID string     `json:"file_unique_id"`      // 该文件的唯一标识符，应该在不同时间和不同机器人中保持不变。不能用于下载或重用该文件。
	Duration     int        `json:"duration"`            // 音频持续时间（以秒为单位），由发送者定义
	Performer    string     `json:"performer,omitempty"` // 可选。音频的表演者，由发送者或音频标签定义
	Title        string     `json:"title,omitempty"`     // 可选。音频的标题，由发送者或音频标签定义
	FileName     string     `json:"file_name,omitempty"` // 可选。发送者定义的原始文件名
	MimeType     string     `json:"mime_type,omitempty"` // 可选。发送者定义的文件 MIME 类型
	FileSize     int64      `json:"file_size,omitempty"` // 可选。文件大小（以字节为单位）。可能大于 2^31，某些编程语言可能在解释时遇到困难/出现静默缺陷。但它最多具有 52 位有效位，因此使用带符号的 64 位整数或双精度浮点类型来存储此值是安全的。
	Thumbnail    *PhotoSize `json:"thumb,omitempty"`     // 可选。音乐文件所属的专辑封面的缩略图
}

// Document 表示通用文件（与照片、语音消息和音频文件相对）。
type Document struct {
	FileID       string     `json:"file_id"`             // 该文件的标识符，可以用于下载或重用该文件
	FileUniqueID string     `json:"file_unique_id"`      // 该文件的唯一标识符，应该在不同时间和不同机器人中保持不变。不能用于下载或重用该文件。
	Thumbnail    *PhotoSize `json:"thumb,omitempty"`     // 可选。发送者定义的文档缩略图
	FileName     string     `json:"file_name,omitempty"` // 可选。发送者定义的原始文件名
	MimeType     string     `json:"mime_type,omitempty"` // 可选。发送者定义的文件 MIME 类型
	FileSize     int64      `json:"file_size,omitempty"` // 可选。文件大小（以字节为单位）。可能大于 2^31，某些编程语言可能在解释时遇到困难/出现静默缺陷。但它最多具有 52 位有效位，因此使用带符号的 64 位整数或双精度浮点类型来存储此值是安全的。
}

// Story 表示有关聊天中转发故事的消息。目前不包含任何信息。
type Story struct {
	Chat Chat  `json:"chat"` // 发布故事的聊天
	ID   int64 `json:"id"`   // 聊天中故事的唯一标识符
}

// Video 表示视频文件。
type Video struct {
	FileID       string     `json:"file_id"`             // 该文件的标识符，可以用于下载或重用该文件
	FileUniqueID string     `json:"file_unique_id"`      // 该文件的唯一标识符，应该在不同时间和不同机器人中保持不变。不能用于下载或重用该文件。
	Width        int        `json:"width"`               // 视频宽度，由发送者定义
	Height       int        `json:"height"`              // 视频高度，由发送者定义
	Duration     int        `json:"duration"`            // 视频持续时间（以秒为单位），由发送者定义
	Thumbnail    *PhotoSize `json:"thumb,omitempty"`     // 可选。视频缩略图
	FileName     string     `json:"file_name,omitempty"` // 可选。发送者定义的原始文件名
	MimeType     string     `json:"mime_type,omitempty"` // 可选。发送者定义的文件 MIME 类型
	FileSize     int64      `json:"file_size,omitempty"` // 可选。文件大小（以字节为单位）。可能大于 2^31，某些编程语言可能在解释时遇到困难/出现静默缺陷。但它最多具有 52 位有效位，因此使用带符号的 64 位整数或双精度浮点类型来存储此值是安全的。
}

// VideoNote 表示视频消息（自 v.4.0 起在 Telegram 应用中可用）。
type VideoNote struct {
	FileID       string     `json:"file_id"`             // 该文件的标识符，可以用于下载或重用该文件
	FileUniqueID string     `json:"file_unique_id"`      // 该文件的唯一标识符，应该在不同时间和不同机器人中保持不变。不能用于下载或重用该文件。
	Length       int        `json:"length"`              // 视频宽度和高度（视频消息的直径），由发送者定义
	Duration     int        `json:"duration"`            // 视频持续时间（以秒为单位），由发送者定义
	Thumbnail    *PhotoSize `json:"thumb,omitempty"`     // 可选。视频缩略图
	FileSize     int64      `json:"file_size,omitempty"` // 可选。文件大小（以字节为单位）
}

// Voice 表示语音消息。
type Voice struct {
	FileID       string `json:"file_id"`             // 该文件的标识符，可以用于下载或重用该文件
	FileUniqueID string `json:"file_unique_id"`      // 该文件的唯一标识符，应该在不同时间和不同机器人中保持不变。不能用于下载或重用该文件。
	Duration     int    `json:"duration"`            // 音频持续时间（以秒为单位），由发送者定义
	MimeType     string `json:"mime_type,omitempty"` // 可选。发送者定义的文件 MIME 类型
	FileSize     int64  `json:"file_size,omitempty"` // 可选。文件大小（以字节为单位）。可能大于 2^31，某些编程语言可能在解释时遇到困难/出现静默缺陷。但它最多具有 52 位有效位，因此使用带符号的 64 位整数或双精度浮点类型来存储此值是安全的。
}

// Contact 表示电话号码联系人。
type Contact struct {
	PhoneNumber string `json:"phone_number"`        // 联系人的电话号码
	FirstName   string `json:"first_name"`          // 联系人的名字
	LastName    string `json:"last_name,omitempty"` // 可选。联系人的姓氏
	UserID      int64  `json:"user_id,omitempty"`   // 可选。联系人在 Telegram 中的用户标识符。此数字可能具有超过 32 位的有效位，某些编程语言可能在解释时遇到困难/出现静默缺陷。但它最多具有 52 位有效位，因此使用 64 位整数或双精度浮点类型来存储此标识符是安全的。
	VCard       string `json:"vcard,omitempty"`     // 可选。以 vCard 形式提供的有关联系人的附加数据
}

// Dice 表示一个显示随机值的动画表情。
type Dice struct {
	Emoji string `json:"emoji"` // 骰子投掷动画所依据的表情
	Value int    `json:"value"` // 骰子的值，1-6 对于“🎲”、“🎯”和“🎳”基础表情，1-5 对于“🏀”和“⚽”基础表情，1-64 对于“🎰”基础表情
}

// PollOption 包含有关投票选项的信息。
type PollOption struct {
	Text         string           `json:"text"`          // 选项文本，1-100 个字符
	TextEntities []*MessageEntity `json:"text_entities"` // 可选。出现在选项文本中的特殊实体。目前，仅允许自定义表情实体出现在投票选项文本中
	VoterCount   int              `json:"voter_count"`   // 投票该选项的用户数量
}

// InputPollOption 包含发送投票的一个选项的信息。
type InputPollOption struct {
	Text          string           `json:"text"`            // 选项文本，1-100 个字符
	TextParseMode string           `json:"text_parse_mode"` // 可选。解析文本中实体的模式。有关详细信息，请参见格式选项。目前，仅允许自定义表情实体出现在投票选项文本中
	TextEntities  []*MessageEntity `json:"text_entities"`   // 可选。出现在投票选项文本中的特殊实体的 JSON 序列化列表。可以替代 text_parse_mode 指定
}

// PollAnswer 表示用户在非匿名投票中的答案。
type PollAnswer struct {
	PollID    string `json:"poll_id"`              // 唯一投票标识符
	VoterChat *Chat  `json:"voter_chat,omitempty"` // 可选。如果投票者是匿名的，则表示更改投票答案的聊天
	User      *User  `json:"user,omitempty"`       // 可选。如果投票者不是匿名的，则表示更改投票答案的用户
	OptionIDs []int  `json:"option_ids"`           // 用户选择的答案选项的基于 0 的标识符。如果用户撤回投票，可能为空。
}

// Poll 包含有关投票的信息。
type Poll struct {
	ID                    string           `json:"id"`                             // 唯一投票标识符
	Question              string           `json:"question"`                       // 投票问题，1-300 个字符
	QuestionEntities      []*MessageEntity `json:"question_entities"`              // 可选。出现在问题中的特殊实体。目前，仅允许自定义表情实体出现在投票问题中
	Options               []PollOption     `json:"options"`                        // 投票选项列表
	TotalVoterCount       int              `json:"total_voter_count"`              // 投票总用户数量
	IsClosed              bool             `json:"is_closed"`                      // 如果投票已关闭，则为真
	IsAnonymous           bool             `json:"is_anonymous"`                   // 如果投票是匿名的，则为真
	Type                  string           `json:"type"`                           // 投票类型，目前可以是“regular”或“quiz”
	AllowsMultipleAnswers bool             `json:"allows_multiple_answers"`        // 如果投票允许多个答案，则为真
	CorrectOptionID       int              `json:"correct_option_id,omitempty"`    // 可选。正确答案选项的基于 0 的标识符。仅在测验模式的投票中可用，且投票已关闭，或由机器人发送（而非转发）至与机器人的私聊中。
	Explanation           string           `json:"explanation,omitempty"`          // 可选。当用户选择错误答案或点击测验式投票中的灯泡图标时显示的文本，0-200 个字符
	ExplanationEntities   []*MessageEntity `json:"explanation_entities,omitempty"` // 可选。出现在解释中的特殊实体，如用户名、URL、机器人命令等。
	OpenPeriod            int              `json:"open_period,omitempty"`          // 可选。投票创建后将处于活动状态的时间（以秒为单位）
	CloseDate             int64            `json:"close_date,omitempty"`           // 可选。投票将自动关闭的时间点（Unix 时间戳）
}

// Location 表示地图上的一个点。
type Location struct {
	Longitude            float64 `json:"longitude"`                        // 由发送者定义的经度
	Latitude             float64 `json:"latitude"`                         // 由发送者定义的纬度
	HorizontalAccuracy   float64 `json:"horizontal_accuracy,omitempty"`    // 可选。位置的不确定性半径，以米为单位；0-1500
	LivePeriod           int     `json:"live_period,omitempty"`            // 可选。相对于消息发送日期的位置更新时间；以秒为单位。仅适用于活动的实时位置。
	Heading              int     `json:"heading,omitempty"`                // 可选。用户移动的方向，以度为单位；1-360。仅适用于活动的实时位置。
	ProximityAlertRadius int     `json:"proximity_alert_radius,omitempty"` // 可选。关于接近其他聊天成员的警报的最大距离，以米为单位。仅适用于发送的实时位置。
}

// Venue 表示一个场地。
type Venue struct {
	Location        Location `json:"location"`                    // 场地位置。不能是实时位置
	Title           string   `json:"title"`                       // 场地名称
	Address         string   `json:"address"`                     // 场地地址
	FoursquareID    string   `json:"foursquare_id,omitempty"`     // 可选。场地的 Foursquare 标识符
	FoursquareType  string   `json:"foursquare_type,omitempty"`   // 可选。场地的 Foursquare 类型。（例如，“arts_entertainment/default”、“arts_entertainment/aquarium”或“food/ice cream”。）
	GooglePlaceID   string   `json:"google_place_id,omitempty"`   // 可选。场地的 Google Places 标识符
	GooglePlaceType string   `json:"google_place_type,omitempty"` // 可选。场地的 Google Places 类型。（请参阅支持的类型。）
}

// WebAppData 描述从 Web 应用发送到机器人的数据。
type WebAppData struct {
	Data       string `json:"data"`        // 数据。请注意，恶意客户端可以在此字段中发送任意数据。
	ButtonText string `json:"button_text"` // 从中打开 Web 应用的 web_app 按钮的文本。请注意，恶意客户端可以在此字段中发送任意数据。
}

// ProximityAlertTriggered 表示服务消息的内容，每当聊天中的用户触发由其他用户设置的接近警报时发送。
type ProximityAlertTriggered struct {
	Traveler User `json:"traveler"` // 触发警报的用户
	Watcher  User `json:"watcher"`  // 设置警报的用户
	Distance int  `json:"distance"` // 用户之间的距离
}

// MessageAutoDeleteTimerChanged 表示有关自动删除计时器设置更改的服务消息。
type MessageAutoDeleteTimerChanged struct {
	MessageAutoDeleteTime int `json:"message_auto_delete_time"` // 聊天中消息的新自动删除时间；以秒为单位
}

// ChatBoostAdded 表示用户提升聊天的服务消息。
type ChatBoostAdded struct {
	BoostCount int `json:"boost_count"` // 用户添加的提升次数
}

// BackgroundFill 描述根据所选颜色填充背景的方式。目前，它可以是以下之一
type BackgroundFill struct {
	BackgroundFillSolid
	BackgroundFillGradient
	BackgroundFillFreeformGradient
}

// BackgroundFillSolid 使用所选颜色填充。
type BackgroundFillSolid struct {
	Type  string `json:"type"`  // 背景填充的类型，始终为“solid”
	Color int    `json:"color"` // 背景填充的颜色，以 RGB24 格式表示
}

// BackgroundFillGradient 根据所选颜色自动填充。
type BackgroundFillGradient struct {
	Type          string `json:"type"`           // 背景填充的类型，始终为“gradient”
	TopColor      int    `json:"top_color"`      // 渐变的顶部颜色，以 RGB24 格式表示
	BottomColor   int    `json:"bottom_color"`   // 渐变的底部颜色，以 RGB24 格式表示
	RotationAngle int    `json:"rotation_angle"` // 背景填充的顺时针旋转角度，以度为单位；0-359
}

// BackgroundFillFreeformGradient 根据所选颜色自动填充。
type BackgroundFillFreeformGradient struct {
	Type   string `json:"type"`   // 背景填充的类型，始终为“freeform_gradient”
	Colors []int  `json:"colors"` // 用于生成自由形式渐变的 3 或 4 种基础颜色的列表，采用 RGB24 格式
}

// BackgroundType 描述背景的类型。目前，它可以是以下之一
type BackgroundType struct {
	BackgroundTypeFill
	BackgroundTypeWallpaper
	BackgroundTypePattern
	BackgroundTypeChatTheme
}

// BackgroundTypeFill 根据所选颜色自动填充。
type BackgroundTypeFill struct {
	Type             string         `json:"type"`               // 背景的类型，始终为“fill”
	Fill             BackgroundFill `json:"fill"`               // 背景填充
	DarkThemeDimming int            `json:"dark_theme_dimming"` // 在黑暗主题中背景的暗淡程度，作为百分比；0-100
}

// BackgroundTypeWallpaper JPEG 格式的壁纸。
type BackgroundTypeWallpaper struct {
	Type             string   `json:"type"`               // 背景的类型，始终为“wallpaper”
	Document         Document `json:"document"`           // 包含壁纸的文档
	DarkThemeDimming int      `json:"dark_theme_dimming"` // 在黑暗主题中背景的暗淡程度，作为百分比；0-100
	IsBlurred        bool     `json:"is_blurred"`         // 可选。如果壁纸被缩小以适应 450x450 的方形并且经过半径为 12 的模糊处理，则为真
	IsMoving         bool     `json:"is_moving"`          // 可选。如果背景在设备倾斜时稍微移动，则为真
}

// BackgroundTypePattern PNG 或 TGV（以 MIME 类型“application/x-tgwallpattern”压缩的 SVG 子集）图案，与用户选择的背景填充组合。
type BackgroundTypePattern struct {
	Type       string         `json:"type"`        // 背景的类型，始终为“pattern”
	Document   Document       `json:"document"`    // 包含图案的文档
	Fill       BackgroundFill `json:"fill"`        // 与图案组合的背景填充
	Intensity  int            `json:"intensity"`   // 图案在填充背景上显示时的强度；0-100
	IsInverted bool           `json:"is_inverted"` // 可选。如果背景填充仅应用于图案本身，则为真。在这种情况下，所有其他像素为黑色。仅适用于黑暗主题
	IsMoving   bool           `json:"is_moving"`   // 可选。如果背景在设备倾斜时稍微移动，则为真
}

// BackgroundTypeChatTheme 直接来自内置聊天主题。
type BackgroundTypeChatTheme struct {
	Type      string `json:"type"`       // 背景的类型，始终为“chat_theme”
	ThemeName string `json:"theme_name"` // 聊天主题的名称，通常是一个表情符号
}

// ChatBackground 表示一个聊天背景。
type ChatBackground struct {
	Type BackgroundType `json:"type"` // 用户添加的提升数量
}

// ForumTopicCreated 表示关于在聊天中创建新论坛主题的服务消息。
type ForumTopicCreated struct {
	Name              string `json:"name"`                           // 主题名称
	IconColor         int    `json:"icon_color"`                     // 主题图标的颜色，以 RGB 格式表示
	IconCustomEmojiID string `json:"icon_custom_emoji_id,omitempty"` // 可选。作为主题图标显示的自定义表情的唯一标识符
}

// ForumTopicClosed 表示关于在聊天中关闭论坛主题的服务消息。目前不包含任何信息。
type ForumTopicClosed struct {
}

// ForumTopicEdited 表示关于编辑论坛主题的服务消息。
type ForumTopicEdited struct {
	Name              string `json:"name,omitempty"`                 // 可选。如果主题被编辑，则为新主题名称
	IconCustomEmojiID string `json:"icon_custom_emoji_id,omitempty"` // 可选。如果主题图标被编辑，则为新的自定义表情标识符；如果图标被移除，则为空字符串
}

// ForumTopicReopened 表示关于在聊天中重新打开论坛主题的服务消息。目前不包含任何信息。
type ForumTopicReopened struct {
}

// GeneralForumTopicHidden 表示关于在聊天中隐藏一般论坛主题的服务消息。目前不包含任何信息。
type GeneralForumTopicHidden struct {
}

// GeneralForumTopicUnhidden 表示关于在聊天中取消隐藏一般论坛主题的服务消息。目前不包含任何信息。
type GeneralForumTopicUnhidden struct {
}

// SharedUser 包含关于通过 KeyboardButtonRequestUser 按钮与机器人共享的用户的信息。
type SharedUser struct {
	UserId    int64        `json:"user_id"`              // 共享用户的标识符。此数字可能具有超过 32 位的有效位，某些编程语言可能在解释时遇到困难/出现静默缺陷。但它最多具有 52 位有效位，因此使用 64 位整数或双精度浮点类型来存储这些标识符是安全的。机器人可能无法访问用户，因此可能无法使用该标识符，除非用户已经通过其他方式为机器人所知。
	FirstName string       `json:"first_name,omitempty"` // 可选。如果用户的名字是机器人请求的，则为用户的名字
	LastName  string       `json:"last_name,omitempty"`  // 可选。如果用户的姓氏是机器人请求的，则为用户的姓氏
	Username  string       `json:"username,omitempty"`   // 可选。如果用户的用户名是机器人请求的，则为用户的用户名
	Photo     []*PhotoSize `json:"photo,omitempty"`      // 可选。如果机器人请求了用户的照片，则为可用的聊天照片的大小
}

// UsersShared 包含通过 KeyboardButtonRequestUsers 按钮与机器人共享的用户标识符的信息。
type UsersShared struct {
	RequestID int           `json:"request_id"` // 请求的标识符
	Users     []*SharedUser `json:"users"`      // 与机器人共享的用户的信息。
}

// ChatShared 包含通过 KeyboardButtonRequestChat 按钮与机器人共享的聊天标识符的信息。
type ChatShared struct {
	RequestID int          `json:"request_id"`         // 请求的标识符
	ChatID    int64        `json:"chat_id"`            // 共享聊天的标识符。此数字可能具有超过 32 位的有效位，某些编程语言可能在解释时遇到困难/出现静默缺陷。但它最多具有 52 位有效位，因此使用 64 位整数或双精度浮点类型来存储该标识符是安全的。机器人可能无法访问聊天，因此可能无法使用该标识符，除非聊天已经通过其他方式为机器人所知。
	Title     string       `json:"title,omitempty"`    // 可选。如果聊天的标题是机器人请求的，则为聊天的标题。
	Username  string       `json:"username,omitempty"` // 可选。如果聊天的用户名是机器人请求的并且可用，则为聊天的用户名。
	Photo     []*PhotoSize `json:"photo,omitempty"`    // 可选。如果机器人请求了聊天的照片，则为可用的聊天照片的大小
}

// WriteAccessAllowed 表示关于用户允许机器人发送消息的服务消息
// 在将机器人添加到附件菜单或从链接启动 Web 应用后。
type WriteAccessAllowed struct {
	FromRequest        bool   `json:"from_request,omitempty"`         // 可选。如果访问是在用户接受 Web 应用程序的明确请求后授予的，则为真
	WebAppName         string `json:"web_app_name,omitempty"`         // 可选。从链接启动的 Web 应用的名称
	FromAttachmentMenu bool   `json:"from_attachment_menu,omitempty"` // 可选。如果访问是在机器人被添加到附件或侧边菜单时授予的，则为真
}

// VideoChatScheduled 表示关于在聊天中安排视频聊天的服务消息。
type VideoChatScheduled struct {
	StartDate int64 `json:"start_date"` // 视频聊天应该由聊天管理员启动的时间点（Unix 时间戳）
}

// VideoChatStarted 表示关于在聊天中开始视频聊天的服务消息。目前不包含任何信息。
type VideoChatStarted struct{}

// VideoChatEnded 表示关于在聊天中结束视频聊天的服务消息。
type VideoChatEnded struct {
	Duration int `json:"duration"` // 视频聊天持续时间（以秒为单位）
}

// VideoChatParticipantsInvited 表示关于新成员被邀请参加视频聊天的服务消息。
type VideoChatParticipantsInvited struct {
	Users []User `json:"users"` // 被邀请参加视频聊天的新成员
}

// GiveawayCreated 表示关于创建计划中的赠品的服务消息。目前不包含任何信息。
type GiveawayCreated struct {
}

// Giveaway 表示关于计划中的赠品的消息。
type Giveaway struct {
	Chats                         []Chat   `json:"chats"`                                      // 用户必须加入的聊天列表，以参与赠品
	WinnersSelectionDate          int64    `json:"winners_selection_date"`                     // 选择赠品获胜者的时间点（Unix 时间戳）
	WinnerCount                   int      `json:"winner_count"`                               // 应该被选为赠品获胜者的用户数量
	OnlyNewMembers                bool     `json:"only_new_members,omitempty"`                 // 可选。如果只有在赠品开始后加入聊天的用户才有资格获胜，则为真
	HasPublicWinners              bool     `json:"has_public_winners,omitempty"`               // 可选。如果赠品获胜者的列表将对所有人可见，则为真
	PrizeDescription              string   `json:"prize_description,omitempty"`                // 可选。额外赠品的描述
	CountryCodes                  []string `json:"country_codes,omitempty"`                    // 可选。表示符合赠品资格的用户必须来自的两位字母 ISO 3166-1 alpha-2 国家代码的列表。如果为空，则所有用户都可以参与赠品。具有在 Fragment 上购买的电话号码的用户始终可以参加赠品。
	PremiumSubscriptionMonthCount int      `json:"premium_subscription_month_count,omitempty"` // 可选。赠品中赢得的 Telegram Premium 订阅将持续的月份数量
}

// GiveawayWinners 表示关于公开获胜者的赠品完成消息。
type GiveawayWinners struct {
	Chat                          Chat   `json:"chats"`                                      // 创建赠品的聊天
	GiveawayMessageId             int    `json:"giveaway_message_id"`                        // 聊天中赠品消息的标识符
	WinnersSelectionDate          int64  `json:"winners_selection_date"`                     // 选择赠品获胜者的时间点（Unix 时间戳）
	WinnerCount                   int    `json:"winner_count"`                               // 赠品中的总获胜者数量
	Winners                       []User `json:"winners"`                                    // 最多 100 位赠品获胜者的列表
	AdditionalChatCount           int    `json:"additional_chat_count,omitempty"`            // 可选。用户必须加入的其他聊天的数量，以符合赠品资格
	PremiumSubscriptionMonthCount int    `json:"premium_subscription_month_count,omitempty"` // 可选。赠品中赢得的 Telegram Premium 订阅将持续的月份数量
	UnclaimedPrizeCount           int    `json:"unclaimed_prize_count,omitempty"`            // 可选。未分配的奖品数量
	OnlyNewMembers                bool   `json:"only_new_members,omitempty"`                 // 可选。如果只有在赠品开始后加入聊天的用户才有资格获胜，则为真
	WasRefunded                   bool   `json:"was_refunded,omitempty"`                     // 可选。如果赠品因退款而取消，则为真
	PrizeDescription              string `json:"prize_description,omitempty"`                // 可选。额外赠品的描述
}

// GiveawayCompleted 表示关于没有公开获胜者的赠品完成的服务消息。
type GiveawayCompleted struct {
	WinnerCount         int      `json:"winner_count"`                    // 赠品中的获胜者数量
	UnclaimedPrizeCount int      `json:"unclaimed_prize_count,omitempty"` // 可选。未分配的奖品数量
	GiveawayMessage     *Message `json:"giveaway_message,omitempty"`      // 可选。如果赠品未被删除，则为完成的赠品消息
}

// LinkPreviewOptions 描述用于链接预览生成的选项。
type LinkPreviewOptions struct {
	IsDisabled       bool   `json:"is_disabled,omitempty"`        // 可选。如果链接预览被禁用，则为真
	Url              string `json:"url,omitempty"`                // 可选。用于链接预览的 URL。如果为空，则将使用消息文本中找到的第一个 URL
	PreferSmallMedia bool   `json:"prefer_small_media,omitempty"` // 可选。如果链接预览中的媒体应缩小，则为真；如果未明确指定 URL 或不支持媒体大小更改，则被忽略
	PreferLargeMedia bool   `json:"prefer_large_media,omitempty"` // 可选。如果链接预览中的媒体应放大，则为真；如果未明确指定 URL 或不支持媒体大小更改，则被忽略
	ShowAboveText    bool   `json:"show_above_text,omitempty"`    // 可选。如果链接预览必须显示在消息文本上方，则为真；否则，链接预览将显示在消息文本下方
}

// UserProfilePhotos 表示用户的个人资料照片。
type UserProfilePhotos struct {
	TotalCount int           `json:"total_count"` // 目标用户拥有的个人资料照片的总数
	Photos     [][]PhotoSize `json:"photos"`      // 请求的个人资料照片（每张最多 4 种大小）
}

// File 表示一个可以下载的文件。
// 文件可以通过链接 https://api.telegram.org/file/bot<token>/<file_path> 下载。保证
// 链接至少有效 1 小时。
// 当链接过期时，可以通过调用 getFile 请求新的链接。
// 注意：最大下载文件大小为 20 MB
type File struct {
	FileID       string `json:"file_id"`             // 此文件的标识符，可用于下载或重用该文件
	FileUniqueID string `json:"file_unique_id"`      // 此文件的唯一标识符，应该在不同时间和不同机器人中保持不变。不能用于下载或重用该文件。
	FileSize     int64  `json:"file_size,omitempty"` // 可选。文件大小（以字节为单位）。可能超过 2^31，某些编程语言可能在解释时遇到困难/出现静默缺陷。但它最多具有 52 位有效位，因此使用带符号的 64 位整数或双精度浮点类型来存储此值是安全的。
	FilePath     string `json:"file_path,omitempty"` // 可选。文件路径。使用 https://api.telegram.org/file/bot<token>/<file_path> 获取文件。
}

// ReplyKeyboardMarkup 表示一个自定义键盘，包含回复选项（有关详细信息和示例，请参见机器人简介）。
type ReplyKeyboardMarkup struct {
	Keyboard              [][]KeyboardButton `json:"keyboard"`                          // 按钮行的数组，每行由 KeyboardButton 对象数组表示
	IsPersistent          bool               `json:"is_persistent,omitempty"`           // 可选。请求客户端在常规键盘隐藏时始终显示此键盘。默认值为 false，此时自定义键盘可以被隐藏，并通过键盘图标打开。
	ResizeKeyboard        bool               `json:"resize_keyboard,omitempty"`         // 可选。请求客户端根据最佳适配垂直调整键盘大小（例如，如果只有两行按钮，则使键盘变小）。默认值为 false，此时自定义键盘的高度始终与应用程序的标准键盘相同。
	OneTimeKeyboard       bool               `json:"one_time_keyboard,omitempty"`       // 可选。请求客户端在使用此键盘后隐藏键盘。键盘仍然可用，但客户端将自动在聊天中显示常规字母键盘 - 用户可以在输入字段中按特殊按钮再次查看自定义键盘。默认值为 false。
	InputFieldPlaceholder string             `json:"input_field_placeholder,omitempty"` // 可选。当键盘处于活动状态时，在输入字段中显示的占位符；1-64 个字符
	Selective             bool               `json:"selective,omitempty"`               // 可选。如果只想对特定用户显示此键盘，请使用此参数。目标：1）在消息对象的文本中提到的用户；2）如果机器人的消息是回复（具有 reply_to_message_id），则是原始消息的发送者。示例：用户请求更改机器人的语言，机器人回复请求并显示选择新语言的键盘。其他用户在群组中看不到此键盘。
}

// KeyboardButton 表示回复键盘的一个按钮。
// 对于简单文本按钮，可以使用字符串代替此对象来指定按钮的文本。
// 可选字段 web_app、request_contact、request_location 和 request_poll 是互斥的。
// 注意：request_contact 和 request_location 选项仅在 2016 年 4 月 9 日之后发布的 Telegram 版本中有效。
// 较旧的客户端将显示不支持的消息。
// 注意：request_poll 选项仅在 2020 年 1 月 23 日之后发布的 Telegram 版本中有效。
// 较旧的客户端将显示不支持的消息。
// 注意：web_app 选项仅在 2022 年 4 月 16 日之后发布的 Telegram 版本中有效。
// 较旧的客户端将显示不支持的消息。
type KeyboardButton struct {
	Text            string                      `json:"text"`                       // 按钮的文本。如果没有使用任何可选字段，则在按钮被按下时以消息的形式发送
	RequestUsers    *KeyboardButtonRequestUsers `json:"request_users,omitempty"`    // 可选。如果指定，按下按钮时将打开合适用户的列表。点击任何用户将以“users_shared”服务消息的形式将其标识符发送给机器人。仅在私聊中可用。
	RequestChat     *KeyboardButtonRequestChat  `json:"request_chat,omitempty"`     // 可选。如果指定，按下按钮时将打开合适聊天的列表。点击一个聊天将以“chat_shared”服务消息的形式将其标识符发送给机器人。仅在私聊中可用。
	RequestContact  bool                        `json:"request_contact,omitempty"`  // 可选。如果为 true，用户的电话号码将在按钮被按下时作为联系人发送。仅在私聊中可用。
	RequestLocation bool                        `json:"request_location,omitempty"` // 可选。如果为 true，用户的当前位置将在按钮被按下时发送。仅在私聊中可用。
	RequestPoll     *KeyboardButtonPollType     `json:"request_poll,omitempty"`     // 可选。如果指定，用户将被要求创建一个投票并在按钮被按下时发送给机器人。仅在私聊中可用。
	WebApp          *WebAppInfo                 `json:"web_app,omitempty"`          // 可选。如果指定，描述的 Web 应用将在按钮被按下时启动。Web 应用能够发送“web_app_data”服务消息。仅在私聊中可用。
}

// ReplyKeyboardRemove 在收到带有此对象的消息时，
// Telegram 客户端将移除当前自定义键盘并显示默认字母键盘。
// 默认情况下，自定义键盘将在新的键盘被机器人发送之前显示。
// 一次性键盘在用户按下按钮后将立即隐藏（见 ReplyKeyboardMarkup）。
type ReplyKeyboardRemove struct {
	RemoveKeyboard bool `json:"remove_keyboard"`     // 请求客户端移除自定义键盘（用户将无法调出此键盘；如果想要隐藏键盘但保持其可访问性，请在 ReplyKeyboardMarkup 中使用 one_time_keyboard）
	Selective      bool `json:"selective,omitempty"` // 可选。如果您想仅对特定用户移除键盘，请使用此参数。目标：1）在消息对象的文本中提到的用户；2）如果机器人的消息是回复（具有 reply_to_message_id），则是原始消息的发送者。示例：用户在投票中，机器人在回复投票时返回确认消息并为该用户移除键盘，同时对尚未投票的用户仍显示投票选项的键盘。
}

// InlineKeyboardMarkup 表示一个内联键盘，出现在其所属的消息旁边。
// 注意：这仅适用于 2016 年 4 月 9 日之后发布的 Telegram 版本。
// 较旧的客户端将显示不支持的消息。
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"` // 按钮行的数组，每行由 InlineKeyboardButton 对象数组表示

}

// InlineKeyboardButton 表示内联键盘的一个按钮。您必须使用恰好一个可选字段。
type InlineKeyboardButton struct {
	Text                         string                       `json:"text"`                                       // 按钮上的标签文本
	URL                          string                       `json:"url,omitempty"`                              // 可选。按下按钮时要打开的 HTTP 或 tg:// URL。链接 tg://user?id=<user_id> 可以用于根据其隐私设置提及用户的 ID，而无需使用用户名。
	CallbackData                 string                       `json:"callback_data,omitempty"`                    // 可选。按下按钮时将启动的 Web 应用的描述。Web 应用能够使用 answerWebAppQuery 方法代表用户发送任意消息。仅在用户与机器人之间的私聊中可用。
	WebApp                       *WebAppInfo                  `json:"web_app,omitempty"`                          // 可选。用于自动授权用户的 HTTPS URL。可用作 Telegram 登录小部件的替代品。
	LoginURL                     *LoginURL                    `json:"login_url,omitempty"`                        // 可选。按下按钮时将发送到机器人的回调查询中的数据，1-64 字节
	SwitchInlineQuery            *string                      `json:"switch_inline_query,omitempty"`              // 可选。如果设置，按下按钮将提示用户选择其聊天，打开该聊天并在输入字段中插入机器人的用户名和指定的内联查询。可以为空，此时仅插入机器人的用户名。注意：这为用户提供了在与机器人进行私聊时开始使用您的机器人内联模式的简单方法。特别是在与 switch_pm… 操作结合使用时，用户将自动返回到他们切换的聊天，跳过聊天选择屏幕。
	SwitchInlineQueryCurrentChat *string                      `json:"switch_inline_query_current_chat,omitempty"` // 可选。如果设置，按下按钮将插入机器人的用户名和指定的内联查询到当前聊天的输入字段。可以为空，此时仅插入机器人的用户名。这为用户提供了在同一聊天中以内联模式打开您的机器人的快捷方式 - 适合从多个选项中选择内容。
	SwitchInlineQueryChosenChat  *SwitchInlineQueryChosenChat `json:"switch_inline_query_chosen_chat,omitempty"`  // 可选。如果设置，按下按钮将提示用户选择指定类型的聊天，打开该聊天并在输入字段中插入机器人的用户名和指定的内联查询。
	CallbackGame                 *CallbackGame                `json:"callback_game,omitempty"`                    // 可选。按下按钮时将启动的游戏的描述。注意：此类型的按钮必须始终是第一行中的第一个按钮。
	Pay                          bool                         `json:"pay,omitempty"`                              // 可选。如果为 true，发送支付按钮。注意：此类型的按钮必须始终是第一行中的第一个按钮，并且只能在发票消息中使用。
	IsXtelegramToken             bool                         `json:"-"`                                          // 可选。如果为 true，发送支付按钮。注意：此类型的按钮必须始终是第一行中的第一个按钮，并且只能在发票消息中使用。
}

// CallbackQuery 表示来自内联键盘中回调按钮的传入回调查询。
// 如果发起查询的按钮附加在机器人发送的消息上，字段 message 将存在。
// 如果按钮附加在通过机器人发送的消息（以内联模式），字段 inline_message_id 将存在。
// 字段 data 或 game_short_name 中恰好有一个会存在。
// 注意：
// 用户按下回调按钮后，Telegram 客户端将显示进度条，直到您调用 answerCallbackQuery。
// 因此，即使不需要向用户发送通知，也必须通过调用 answerCallbackQuery 进行响应
// （例如，在未指定任何可选参数的情况下）。
type CallbackQuery struct {
	ID              string   `json:"id"`                          // 此查询的唯一标识符
	From            User     `json:"from"`                        // 发送者
	Message         *Message `json:"message,omitempty"`           // 可选。发起查询的回调按钮所在的消息。请注意，如果消息过旧，则消息内容和消息日期将不可用
	InlineMessageID string   `json:"inline_message_id,omitempty"` // 可选。通过机器人以内联模式发送的消息的标识符，该消息发起了查询。
	ChatInstance    string   `json:"chat_instance"`               // 全球唯一标识符，对应于发送回调按钮消息的聊天。可用于游戏中的高分。
	Data            string   `json:"data,omitempty"`              // 可选。与回调按钮关联的数据。请注意，发起查询的消息可能不包含此数据的回调按钮。
	GameShortName   string   `json:"game_short_name,omitempty"`   // 可选。要返回的游戏的短名称，作为游戏的唯一标识符
}

// ForceReply 在收到带有此对象的消息时，
// Telegram 客户端将显示回复界面给用户
// （就像用户手动选择了机器人的消息并点击“回复”一样）。
// 这在创建用户友好的逐步接口时非常有用，而不需要牺牲隐私模式。
// 示例：一个针对群组的投票机器人在隐私模式下运行（仅接收命令，回复其消息和提及）。
// 创建新投票有两种方式：
//
// 1. 向用户解释如何发送带参数的命令（例如 /new poll question answer1 answer2）。
// 也许对于硬核用户来说比较吸引，但缺乏现代化的打磨。
// 2. 引导用户通过逐步过程。
// “请发送您的问题”，“很好，现在让我们添加第一个答案选项”，“太好了。继续添加答案选项，然后在准备好时发送 /done。”
// 最后一个选项显然更有吸引力。
// 如果在机器人的问题中使用 ForceReply，
// 它将接收用户的答案，即使它只接收回复、命令和提及 - 对用户来说没有额外的工作。
type ForceReply struct {
	ForceReply            bool   `json:"force_reply"`                       // 显示回复界面给用户，就像他们手动选择了机器人的消息并点击“回复”一样
	InputFieldPlaceholder string `json:"input_field_placeholder,omitempty"` // 可选。回复活动时在输入字段中显示的占位符；1-64 个字符
	Selective             bool   `json:"selective,omitempty"`               // 可选。如果您想仅强制某些用户回复，请使用此参数。目标：1）在消息对象文本中提到的用户；2）如果机器人的消息是回复（具有 reply_to_message_id），则是原始消息的发送者。
}

// ChatPhoto 表示聊天照片。
type ChatPhoto struct {
	SmallFileID       string `json:"small_file_id"`        // 小型（160x160）聊天照片的文件标识符。此 file_id 仅可用于照片下载，并且仅在照片未更改时有效。
	SmallFileUniqueID string `json:"small_file_unique_id"` // 小型（160x160）聊天照片的唯一文件标识符，应该在时间上和不同的机器人中保持不变。不能用于下载或重用该文件。
	BigFileID         string `json:"big_file_id"`          // 大型（640x640）聊天照片的文件标识符。此 file_id 仅可用于照片下载，并且仅在照片未更改时有效。
	BigFileUniqueID   string `json:"big_file_unique_id"`   // 大型（640x640）聊天照片的唯一文件标识符，应该在时间上和不同的机器人中保持不变。不能用于下载或重用该文件。
}

// ChatInviteLink 表示聊天的邀请链接。
type ChatInviteLink struct {
	InviteLink              string `json:"invite_link"`                          // 邀请链接。如果该链接是由另一个聊天管理员创建，则链接的第二部分将被替换为“…”。
	Creator                 User   `json:"creator"`                              // 链接的创建者
	CreatesJoinRequest      bool   `json:"creates_join_request"`                 // 如果通过该链接加入聊天的用户需要得到聊天管理员的批准，则为真
	IsPrimary               bool   `json:"is_primary"`                           // 如果链接是主要链接，则为真
	IsRevoked               bool   `json:"is_revoked"`                           // 如果链接已被撤销，则为真
	Name                    string `json:"name,omitempty"`                       // 可选。邀请链接名称
	ExpireDate              int64  `json:"expire_date,omitempty"`                // 可选。链接将过期或已过期的时间点（Unix 时间戳）
	MemberLimit             int    `json:"member_limit,omitempty"`               // 可选。通过此邀请链接加入聊天后，可以同时成为聊天成员的最大用户数量；1-99999
	PendingJoinRequestCount int    `json:"pending_join_request_count,omitempty"` // 可选。使用此链接创建的待处理加入请求的数量
}

// ChatAdministratorRights 表示聊天中管理员的权限。
type ChatAdministratorRights struct {
	IsAnonymous         bool `json:"is_anonymous"`                 // 如果用户在聊天中的存在是隐藏的，则为真
	CanManageChat       bool `json:"can_manage_chat"`              // 如果管理员可以访问聊天事件日志、聊天统计、频道消息统计、查看频道成员、查看超级群组中的匿名管理员并忽略慢模式，则为真。由任何其他管理员权限隐含。
	CanDeleteMessages   bool `json:"can_delete_messages"`          // 如果管理员可以删除其他用户的消息，则为真
	CanManageVideoChats bool `json:"can_manage_video_chats"`       // 如果管理员可以管理视频聊天，则为真
	CanRestrictMembers  bool `json:"can_restrict_members"`         // 如果管理员可以限制、禁止或解除禁止聊天成员，则为真
	CanPromoteMembers   bool `json:"can_promote_members"`          // 如果管理员可以添加新的管理员并拥有部分自己的权限，或者直接或间接地降低他提升的管理员的权限，则为真
	CanChangeInfo       bool `json:"can_change_info"`              // 如果用户被允许更改聊天标题、照片和其他设置，则为真
	CanInviteUsers      bool `json:"can_invite_users"`             // 如果用户被允许邀请新用户加入聊天，则为真
	CanPostMessages     bool `json:"can_post_messages,omitempty"`  // 可选。如果管理员可以在频道中发帖，则为真；仅适用于频道
	CanEditMessages     bool `json:"can_edit_messages,omitempty"`  // 可选。如果管理员可以编辑其他用户的消息并可以固定消息，则为真；仅适用于频道
	CanPinMessages      bool `json:"can_pin_messages,omitempty"`   // 可选。如果用户被允许固定消息，则为真；仅适用于群组和超级群组
	CanPostStories      bool `json:"can_post_stories,omitempty"`   // 可选。如果管理员可以在频道中发布故事，则为真；仅适用于频道
	CanEditStories      bool `json:"can_edit_stories,omitempty"`   // 可选。如果管理员可以编辑其他用户发布的故事，则为真；仅适用于频道
	CanDeleteStories    bool `json:"can_delete_stories,omitempty"` // 可选。如果管理员可以删除其他用户发布的故事，则为真；仅适用于频道
	CanManageTopics     bool `json:"can_manage_topics,omitempty"`  // 可选。如果用户被允许创建、重命名、关闭和重新打开论坛主题，则为真；仅适用于超级群组
}

// ChatMemberUpdated 表示聊天成员状态的变化。
type ChatMemberUpdated struct {
	Chat                    Chat            `json:"chat"`                                  // 用户所属的聊天
	From                    User            `json:"from"`                                  // 导致变化的操作执行者
	Date                    int             `json:"date"`                                  // 变化发生的时间（Unix 时间）
	OldChatMember           ChatMember      `json:"old_chat_member"`                       // 聊天成员的先前信息
	NewChatMember           ChatMember      `json:"new_chat_member"`                       // 聊天成员的新信息
	InviteLink              *ChatInviteLink `json:"invite_link,omitempty"`                 // 可选。用户用于加入聊天的邀请链接；仅用于通过邀请链接事件加入。
	ViaJoinRequest          bool            `json:"via_join_request,omitempty"`            // 可选。如果用户在未使用邀请链接的情况下发送直接加入请求并获得管理员批准后加入聊天，则为真
	ViaChatFolderInviteLink bool            `json:"via_chat_folder_invite_link,omitempty"` // 可选。如果用户通过聊天文件夹邀请链接加入聊天，则为真
}

// ChatMember 包含有关聊天中某个成员的信息。目前支持以下六种类型的聊天成员：
type ChatMember struct {
	User                  *User  `json:"user"`                                // 有关用户的信息
	Status                string `json:"status"`                              // 成员在聊天中的状态。可以是“creator”、“administrator”、“member”、“restricted”、“left”或“kicked”
	CustomTitle           string `json:"custom_title,omitempty"`              // 可选。仅限所有者和管理员。此用户的自定义标题
	IsAnonymous           bool   `json:"is_anonymous,omitempty"`              // 可选。仅限所有者和管理员。如果用户在聊天中的存在是隐藏的，则为真
	UntilDate             int64  `json:"until_date,omitempty"`                // 可选。仅限受限和被踢用户。限制将解除的日期；Unix 时间。
	CanBeEdited           bool   `json:"can_be_edited,omitempty"`             // 可选。仅限管理员。如果机器人被允许编辑该用户的管理员权限，则为真。
	CanManageChat         bool   `json:"can_manage_chat,omitempty"`           // 可选。仅限管理员。如果管理员可以访问聊天事件日志、聊天统计、频道消息统计、查看频道成员、查看超级群组中的匿名管理员并忽略慢模式，则为真。由任何其他管理员权限隐含。
	CanPostMessages       bool   `json:"can_post_messages,omitempty"`         // 可选。仅限管理员。如果管理员可以在频道中发帖，则为真；仅适用于频道。
	CanEditMessages       bool   `json:"can_edit_messages,omitempty"`         // 可选。仅限管理员。如果管理员可以编辑其他用户的消息并可以固定消息，则为真；仅适用于频道。
	CanDeleteMessages     bool   `json:"can_delete_messages,omitempty"`       // 可选。仅限管理员。如果管理员可以删除其他用户的消息，则为真。
	CanPostStories        bool   `json:"can_post_stories,omitempty"`          // 可选。如果管理员可以在频道中发布故事，则为真；仅适用于频道。
	CanEditStories        bool   `json:"can_edit_stories,omitempty"`          // 可选。如果管理员可以编辑其他用户发布的故事，则为真；仅适用于频道。
	CanDeleteStories      bool   `json:"can_delete_stories,omitempty"`        // 可选。如果管理员可以删除其他用户发布的故事，则为真；仅适用于频道。
	CanManageVideoChats   bool   `json:"can_manage_video_chats,omitempty"`    // 可选。仅限管理员。如果管理员可以管理视频聊天，则为真。
	CanRestrictMembers    bool   `json:"can_restrict_members,omitempty"`      // 可选。仅限管理员。如果管理员可以限制、禁止或解除禁止聊天成员，则为真。
	CanPromoteMembers     bool   `json:"can_promote_members,omitempty"`       // 可选。仅限管理员。如果管理员可以添加新管理员，并拥有部分自己的权限，或者直接或间接地降低他提升的管理员的权限，则为真。
	CanChangeInfo         bool   `json:"can_change_info,omitempty"`           // 可选。仅限管理员和受限用户。如果用户被允许更改聊天标题、照片和其他设置，则为真。
	CanInviteUsers        bool   `json:"can_invite_users,omitempty"`          // 可选。仅限管理员和受限用户。如果用户被允许邀请新用户加入聊天，则为真。
	CanPinMessages        bool   `json:"can_pin_messages,omitempty"`          // 可选。仅限管理员和受限用户。如果用户被允许固定消息，则为真；仅适用于群组和超级群组。
	IsMember              bool   `json:"is_member,omitempty"`                 // 可选。仅限受限用户。如果用户在请求时是聊天的成员，则为真。
	CanSendMessages       bool   `json:"can_send_messages,omitempty"`         // 可选。仅限受限用户。如果用户被允许发送文本消息、联系人、发票、位置和场所，则为真。
	CanSendAudios         bool   `json:"can_send_audios,omitempty"`           // 可选。仅限受限用户。如果用户被允许发送音频，则为真。
	CanSendDocuments      bool   `json:"can_send_documents,omitempty"`        // 可选。仅限受限用户。如果用户被允许发送文档，则为真。
	CanSendPhotos         bool   `json:"can_send_photos,omitempty"`           // 可选。仅限受限用户。如果用户被允许发送照片，则为真。
	CanSendVideos         bool   `json:"can_send_videos,omitempty"`           // 可选。仅限受限用户。如果用户被允许发送视频，则为真。
	CanSendVideoNotes     bool   `json:"can_send_video_notes,omitempty"`      // 可选。仅限受限用户。如果用户被允许发送视频便条，则为真。
	CanSendVoiceNotes     bool   `json:"can_send_voice_notes,omitempty"`      // 可选。仅限受限用户。如果用户被允许发送语音便条，则为真。
	CanSendPolls          bool   `json:"can_send_polls,omitempty"`            // 可选。仅限受限用户。如果用户被允许发送投票，则为真。
	CanSendOtherMessages  bool   `json:"can_send_other_messages,omitempty"`   // 可选。仅限受限用户。如果用户被允许发送音频、文档、照片、视频、视频便条和语音便条，则为真。
	CanAddWebPagePreviews bool   `json:"can_add_web_page_previews,omitempty"` // 可选。仅限受限用户。如果用户被允许在其消息中添加网页预览，则为真。
	CanManageTopics       bool   `json:"can_manage_topics,omitempty"`         // 可选。仅限管理员和受限用户。如果用户被允许创建、重命名、关闭和重新打开论坛主题，则为真；仅适用于超级群组。
}

// ChatJoinRequest 表示发送给聊天的加入请求。
type ChatJoinRequest struct {
	Chat       Chat            `json:"chat"`                  // 请求发送到的聊天
	From       User            `json:"from"`                  // 发送加入请求的用户
	UserChatId int64           `json:"user_chat_id"`          // 发送加入请求的用户的私人聊天的标识符。此数字可能具有超过 32 位的有效位数，一些编程语言可能在解释时遇到困难/沉默缺陷。但它最多有 52 位有效位，因此使用 64 位整数或双精度浮点类型来存储此标识符是安全的。机器人可以在 24 小时内使用此标识符发送消息，直到加入请求被处理，假设没有其他管理员联系用户。
	Date       int64           `json:"date"`                  // 请求发送的时间（Unix 时间）
	Bio        string          `json:"bio,omitempty"`         // 可选。用户的简介。
	InviteLink *ChatInviteLink `json:"invite_link,omitempty"` // 可选。用户用于发送加入请求的聊天邀请链接
}

// ChatPermissions 描述非管理员用户在聊天中被允许执行的操作。
type ChatPermissions struct {
	CanSendMessages       bool `json:"can_send_messages,omitempty"`         // 可选。如果用户被允许发送文本消息、联系人、位置和场所，则为真
	CanSendAudios         bool `json:"can_send_audios,omitempty"`           // 可选。如果用户被允许发送音频，则为真
	CanSendDocuments      bool `json:"can_send_documents,omitempty"`        // 可选。如果用户被允许发送文档，则为真
	CanSendPhotos         bool `json:"can_send_photos,omitempty"`           // 可选。如果用户被允许发送照片，则为真
	CanSendVideos         bool `json:"can_send_videos,omitempty"`           // 可选。如果用户被允许发送视频，则为真
	CanSendVideoNotes     bool `json:"can_send_video_notes,omitempty"`      // 可选。如果用户被允许发送视频便条，则为真
	CanSendVoiceNotes     bool `json:"can_send_voice_notes,omitempty"`      // 可选。如果用户被允许发送语音便条，则为真
	CanSendPolls          bool `json:"can_send_polls,omitempty"`            // 可选。如果用户被允许发送投票，则为真，这意味着可以发送消息
	CanSendOtherMessages  bool `json:"can_send_other_messages,omitempty"`   // 可选。如果用户被允许发送动画、游戏、贴纸并使用内联机器人，则为真，这意味着可以发送媒体消息
	CanAddWebPagePreviews bool `json:"can_add_web_page_previews,omitempty"` // 可选。如果用户被允许在消息中添加网页预览，则为真，这意味着可以发送媒体消息
	CanChangeInfo         bool `json:"can_change_info,omitempty"`           // 可选。如果用户被允许更改聊天标题、照片和其他设置，则为真。在公共超级群组中被忽略
	CanInviteUsers        bool `json:"can_invite_users,omitempty"`          // 可选。如果用户被允许邀请新用户加入聊天，则为真
	CanPinMessages        bool `json:"can_pin_messages,omitempty"`          // 可选。如果用户被允许固定消息，则为真。在公共超级群组中被忽略
	CanManageTopics       bool `json:"can_manage_topics,omitempty"`         // 可选。如果用户被允许创建论坛主题，则为真。如果省略，则默认为 can_pin_messages 的值
}

// Birthdate //
type Birthdate struct {
	Day   int `json:"day"`            // 用户出生的日期；1-31
	Month int `json:"month"`          // 用户出生的月份；1-12
	Year  int `json:"year,omitempty"` // 可选。用户出生的年份
}

// BusinessIntro //
type BusinessIntro struct {
	Title   string   `json:"title,omitempty"`   // 可选。商业介绍的标题文本
	Message string   `json:"message,omitempty"` // 可选。商业介绍的消息文本
	Sticker *Sticker `json:"sticker,omitempty"` // 可选。商业介绍的贴纸
}

// BusinessLocation //
type BusinessLocation struct {
	Address  string    `json:"address"`            // 商业地址
	Location *Location `json:"location,omitempty"` // 可选。商业位置
}

// BusinessOpeningHoursInterval //
type BusinessOpeningHoursInterval struct {
	OpeningMinute int `json:"opening_minute"` // 一周中开始营业的分钟序列号，从周一开始；0 - 7 * 24 * 60
	ClosingMinute int `json:"closing_minute"` // 一周中结束营业的分钟序列号，从周一开始；0 - 8 * 24 * 60
}

// BusinessOpeningHours //
type BusinessOpeningHours struct {
	TimeZoneName string                          `json:"time_zone_name"` // 定义营业时间的时区的唯一名称
	OpeningHours []*BusinessOpeningHoursInterval `json:"opening_hours"`  // 描述营业时间的时间间隔列表
}

// ChatLocation 表示聊天连接的位置信息。
type ChatLocation struct {
	Location Location `json:"location"` // 超级群组连接的位置。不能是实时位置。
	Address  string   `json:"address"`  // 位置地址；1-64 个字符，由聊天所有者定义
}

// ReactionType 描述反应的类型。目前，它可以是以下之一
type ReactionType struct {
	ReactionTypeEmoji       ReactionTypeEmoji
	ReactionTypeCustomEmoji ReactionTypeCustomEmoji
}

// ReactionTypeEmoji 反应基于表情符号。
type ReactionTypeEmoji struct {
	Type  string `json:"type"`  // 反应的类型，总是“emoji”
	Emoji string `json:"emoji"` // 反应的表情符号。目前，可以是 "👍"、"👎"、"❤" 等等
}

// ReactionTypeCustomEmoji 反应基于自定义表情符号。
type ReactionTypeCustomEmoji struct {
	Type        string `json:"type"`         // 反应的类型，总是“custom_emoji”
	CustomEmoji string `json:"custom_emoji"` // 自定义表情符号的标识符
}

// ReactionCount 表示添加到消息的反应以及它被添加的次数。
type ReactionCount struct {
	Type       ReactionType `json:"type"`        // 反应的类型
	TotalCount int          `json:"total_count"` // 添加反应的次数
}

// MessageReactionUpdated 表示用户对消息反应的变化。
type MessageReactionUpdated struct {
	Chat        Chat           `json:"chat"`                 // 包含用户反应的消息的聊天
	MessageId   int            `json:"message_id"`           // 聊天中消息的唯一标识符
	User        *User          `json:"user,omitempty"`       // 可选。改变反应的用户，如果用户不是匿名的
	ActorChat   *Chat          `json:"actor_chat,omitempty"` // 可选。如果用户是匿名的，则表示反应被改变的聊天
	Date        int64          `json:"date"`                 // 变化发生的日期（Unix 时间）
	OldReaction []ReactionType `json:"old_reaction"`         // 用户设置的先前反应类型列表
	NewReaction []ReactionType `json:"new_reaction"`         // 用户设置的新反应类型列表
}

// MessageReactionCountUpdated 表示带有匿名反应的消息上的反应变化。
type MessageReactionCountUpdated struct {
	Chat      Chat            `json:"chat"`       // 包含消息的聊天
	MessageId int             `json:"message_id"` // 聊天中消息的唯一标识符
	Date      int64           `json:"date"`       // 变化发生的日期（Unix 时间）
	Reactions []ReactionCount `json:"reactions"`  // 消息上存在的反应列表
}

// ForumTopic 表示一个论坛主题。
type ForumTopic struct {
	MessageThreadID   int64  `json:"message_thread_id"`              // 论坛主题的唯一标识符
	Name              string `json:"name"`                           // 主题的名称
	IconColor         int    `json:"icon_color"`                     // 主题图标的 RGB 颜色
	IconCustomEmojiID string `json:"icon_custom_emoji_id,omitempty"` // 可选。作为主题图标显示的自定义表情符号的唯一标识符
}

// BotCommand 表示一个机器人命令。
type BotCommand struct {
	Command     string `json:"command"`     // 命令的文本；1-32 个字符。只能包含小写英文字母、数字和下划线。
	Description string `json:"description"` // 命令的描述；1-256 个字符。
}

// BotCommandScope 表示机器人命令应用的范围。目前支持以下七种范围。
type BotCommandScope struct {
	Type   string `json:"type"`
	ChatID int64  `json:"chat_id"`
	UserID int64  `json:"user_id"`
}

// BotCommandScopeDefault 表示机器人命令的默认范围。
// 默认命令在没有为用户指定更窄范围的命令时使用。
type BotCommandScopeDefault struct {
	Type string `json:"type"` // 范围类型必须是 default
}

// BotCommandScopeAllPrivateChats 表示机器人命令的范围，涵盖所有私聊。
type BotCommandScopeAllPrivateChats struct {
	Type string `json:"type"` // 范围类型，必须是 all_private_chats
}

// BotCommandScopeAllGroupChats 表示机器人命令的范围，涵盖所有群组和超级群组聊天。
type BotCommandScopeAllGroupChats struct {
	Type string `json:"type"` // 范围类型，必须是 all_group_chats
}

// BotCommandScopeAllChatAdministrators 表示机器人命令的范围，涵盖所有群组和超级群组的聊天管理员。
type BotCommandScopeAllChatAdministrators struct {
	Type string `json:"type"` // 范围类型，必须是 all_chat_administrators
}

// BotCommandScopeChat 表示机器人的命令范围，覆盖特定的聊天。
type BotCommandScopeChat struct {
	Type   string `json:"type"`    // 范围类型必须为 chat
	ChatID string `json:"chat_id"` // 目标聊天的唯一标识符或目标超级群组的用户名（格式为 @supergroup username）
}

// BotCommandScopeChatAdministrators 表示机器人的命令范围，覆盖特定群组或超级群组聊天的所有管理员。
type BotCommandScopeChatAdministrators struct {
	Type   string `json:"type"`    // 范围类型，必须为 chat_administrators
	ChatID string `json:"chat_id"` // 目标聊天的唯一标识符或目标超级群组的用户名（格式为 @supergroup username）
}

// BotCommandScopeChatMember 表示机器人的命令范围，覆盖群组或超级群组聊天的特定成员。
type BotCommandScopeChatMember struct {
	Type   string `json:"type"`    // 范围类型，必须为 chat_member
	ChatID string `json:"chat_id"` // 目标聊天的唯一标识符或目标超级群组的用户名（格式为 @supergroup username）
	UserID int64  `json:"user_id"` // 目标用户的唯一标识符
}

// BotName 表示机器人的名称。
type BotName struct {
	Name string `json:"name"` // 机器人的名称
}

// BotDescription 表示机器人的描述。
type BotDescription struct {
	Description string `json:"description"` // 机器人的描述
}

// BotShortDescription 表示机器人的简短描述。
type BotShortDescription struct {
	ShortDescription string `json:"short_description"` // 机器人的简短描述
}

// MenuButton 描述了私聊中的机器人的菜单按钮。
// 如果私聊中设置了除 MenuButtonDefault 以外的菜单按钮，则在聊天中应用它。
// 否则，将应用默认菜单按钮。
// 默认情况下，菜单按钮打开机器人的命令列表。
type MenuButton struct {
	Type   string      `json:"type"`    // 按钮类型必须为 commands
	Text   string      `json:"text"`    // 按钮上的文本
	WebApp *WebAppInfo `json:"web_app"` // 当用户按下按钮时将启动的 Web 应用的描述。Web 应用将能够使用 answerWebAppQuery 方法代表用户发送任意消息。
}

// MenuButtons 描述了私聊中的机器人的菜单按钮。
// 如果私聊中设置了除 MenuButtonDefault 以外的菜单按钮，则在聊天中应用它。
// 否则，将应用默认菜单按钮。
// 默认情况下，菜单按钮打开机器人的命令列表。
type MenuButtons struct {
	MenuButtonCommands MenuButtonCommands
	MenuButtonWebApp   MenuButtonWebApp
	MenuButtonDefault  MenuButtonDefault
}

// MenuButtonCommands 表示一个菜单按钮，打开机器人的命令列表。
type MenuButtonCommands struct {
	Type string `json:"type"` // 按钮类型必须为 commands
}

// MenuButtonWebApp 表示一个菜单按钮，启动一个 Web 应用。
type MenuButtonWebApp struct {
	Type   string      `json:"type"`    // 按钮类型，必须为 web_app
	Text   string      `json:"text"`    // 按钮上的文本
	WebApp *WebAppInfo `json:"web_app"` // 当用户按下按钮时将启动的 Web 应用的描述。Web 应用将能够使用 answerWebAppQuery 方法代表用户发送任意消息。
}

// MenuButtonDefault 描述没有设置特定值的菜单按钮。
type MenuButtonDefault struct {
	Type string `json:"type"` // 按钮类型必须为 default
}

// ChatBoostSource 描述聊天提升的来源。它可以是以下之一
type ChatBoostSource struct {
	ChatBoostSourcePremium
	ChatBoostSourceGiftCode
	ChatBoostSourceGiveaway
}

// ChatBoostSourcePremium 通过订阅 Telegram Premium 或将 Telegram Premium 订阅赠送给其他用户获得。
type ChatBoostSourcePremium struct {
	Source string `json:"source"` // 提升的来源，总是“premium”
	User   User   `json:"user"`   // 提升聊天的用户
}

// ChatBoostSourceGiftCode 通过创建 Telegram Premium 礼品代码来提升聊天获得。每个这样的代码在相应的 Telegram Premium 订阅期间提升聊天 4 次。
type ChatBoostSourceGiftCode struct {
	Source string `json:"source"` // 提升的来源，总是“gift_code”
	User   User   `json:"user"`   // 为其创建礼品代码的用户
}

// ChatBoostSourceGiveaway 通过创建 Telegram Premium 抽奖来获得。这在相应的 Telegram Premium 订阅期间提升聊天 4 次。
type ChatBoostSourceGiveaway struct {
	Source            string `json:"source"`                 // 提升的来源，总是“giveaway”
	GiveawayMessageId int    `json:"giveaway_message_id"`    // 抽奖中的消息在聊天中的标识符；该消息可能已经被删除。如果消息尚未发送，则可以为 0。
	User              *User  `json:"user,omitempty"`         // 可选。赢得抽奖的用户（如果有）
	IsUnclaimed       bool   `json:"is_unclaimed,omitempty"` // 可选。如果抽奖已完成，但没有用户赢得奖品，则为真
}

// ChatBoost 表示添加到聊天或更改的提升。
type ChatBoost struct {
	BoostID        string          `json:"boost_id"`        // 提升的唯一标识符
	AddDate        int64           `json:"add_date"`        // 聊天被提升的时间点（Unix 时间戳）
	ExpirationDate int64           `json:"expiration_date"` // 提升将自动过期的时间点（Unix 时间戳），除非提升者的 Telegram Premium 订阅延长
	Source         ChatBoostSource `json:"source"`          // 添加的提升的来源
}

// ChatBoostUpdated 表示添加到聊天或更改的提升。
type ChatBoostUpdated struct {
	Chat  Chat      `json:"chat"`  // 被提升的聊天
	Boost ChatBoost `json:"boost"` // 聊天提升的信息
}

// ChatBoostRemoved 表示从聊天中移除的提升。
type ChatBoostRemoved struct {
	Chat       Chat            `json:"chat"`        // 被提升的聊天
	BoostId    string          `json:"boost_id"`    // 提升的唯一标识符
	RemoveDate int64           `json:"remove_date"` // 被移除提升的时间点（Unix 时间戳）
	Source     ChatBoostSource `json:"source"`      // 被移除提升的来源
}

// UserChatBoosts 表示用户添加到聊天的提升。
type UserChatBoosts struct {
	Boosts []ChatBoost `json:"boosts"` // 用户添加到聊天的提升列表
}

// BusinessConnection 描述机器人与商业账户的连接。
type BusinessConnection struct {
	ID         string `json:"id"`           // 商业连接的唯一标识符
	User       User   `json:"user"`         // 创建商业连接的商业账户用户
	UserChatId int64  `json:"user_chat_id"` // 与创建商业连接的用户的私人聊天的标识符。该数字可能有超过 32 个有效位，一些编程语言可能在解释时出现困难/静默缺陷。但它最多有 52 个有效位，因此 64 位整数或双精度浮点类型是安全的以存储该标识符。
	Date       int64  `json:"date"`         // 连接建立的日期（Unix 时间）
	CanReply   bool   `json:"can_reply"`    // 如果机器人可以在过去 24 小时内对活动聊天中的商业账户进行操作，则为真
	IsEnabled  bool   `json:"is_enabled"`   // 如果连接处于活动状态，则为真
}

// BusinessMessagesDeleted 在连接的商业账户中删除消息时接收。
type BusinessMessagesDeleted struct {
	BusinessConnectionId string `json:"business_connection_id"` // 商业连接的唯一标识符
	Chat                 Chat   `json:"chat"`                   // 有关商业账户中的聊天的信息。机器人可能没有访问该聊天或相应用户的权限。
	MessageIds           []int  `json:"message_ids"`            // 在商业账户的聊天中删除的消息的标识符的 JSON 序列化列表
}

// ResponseParameters 是 API 响应中可能返回的各种错误。
type ResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id,omitempty"` // 可选。群组已迁移到具有指定标识符的超级群组。该数字可能有超过 32 个有效位，一些编程语言可能在解释时出现困难/静默缺陷。但它最多有 52 个有效位，因此 64 位整数或双精度浮点类型是安全的以存储该标识符。
	RetryAfter      int   `json:"retry_after,omitempty"`        // 可选。如果超过洪水控制，重复请求之前需要等待的秒数
}

// InputMedia 表示要发送的媒体消息的内容。它应该是以下之一
type InputMedia struct {
	InputMediaAnimation InputMediaAnimation
	InputMediaDocument  InputMediaDocument
	InputMediaAudio     InputMediaAudio
	InputMediaPhoto     InputMediaPhoto
	InputMediaVideo     InputMediaVideo
}

// InputMediaPhoto 表示要发送的照片。
type InputMediaPhoto struct {
	Type                  string           `json:"type"`                               // 结果类型必须为 photo
	Media                 RequestFileData  `json:"media"`                              // 要发送的文件。传递 file_id 以发送在 Telegram 服务器上存在的文件（推荐），传递 HTTP URL 以让 Telegram 从互联网上获取文件，或者传递“attach://<file_attach_name>”以使用 multipart/form-data 上传新文件，名称为 <file_attach_name>。
	Caption               string           `json:"caption,omitempty"`                  // 可选。要发送的照片的标题，0-1024 个字符，解析实体后
	ParseMode             string           `json:"parse_mode,omitempty"`               // 可选。解析照片标题中实体的模式。有关详细信息，请参见 [格式选项](https://core.telegram.org/bots/api#formatting-options)。
	CaptionEntities       []*MessageEntity `json:"caption_entities,omitempty"`         // 可选。出现在标题中的特殊实体列表，可以代替 parse_mode 指定
	ShowCaptionAboveMedia bool             `json:"show_caption_above_media,omitempty"` // 可选。如果标题必须显示在消息媒体上方，则传递 True
	HasSpoiler            bool             `json:"has_spoiler,omitempty"`              // 可选。如果照片需要用剧透动画覆盖，则传递 True
}

// InputMediaVideo 表示要发送的视频。
type InputMediaVideo struct {
	Type                  string           `json:"type"`                               // 结果的类型必须为视频
	Media                 RequestFileData  `json:"media"`                              // 要发送的文件。传递 file_id 以发送在 Telegram 服务器上存在的文件（推荐），传递 HTTP URL 让 Telegram 从互联网获取文件，或传递 “attach://<file_attach_name>” 以使用 multipart/form-data 上传新文件。
	Thumbnail             RequestFileData  `json:"thumb,omitempty"`                    // 可选。发送文件的缩略图；如果服务器支持文件的缩略图生成可以忽略。缩略图应为 JPEG 格式且大小小于 200 kB。缩略图的宽度和高度不得超过 320。如果文件不是通过 multipart/form-data 上传，则忽略。缩略图不能重用，只能作为新文件上传，因此可以传递 “attach://<file_attach_name>” 如果缩略图是通过 multipart/form-data 上传的。
	Caption               string           `json:"caption,omitempty"`                  // 可选。要发送的视频的说明，0-1024 个字符，经过实体解析
	ParseMode             string           `json:"parse_mode,omitempty"`               // 可选。视频说明中解析实体的模式。有关更多详细信息，请参见 [格式选项](https://core.telegram.org/bots/api#formatting-options)。
	CaptionEntities       []*MessageEntity `json:"caption_entities,omitempty"`         // 可选。出现在说明中的特殊实体列表，可以替代 parse_mode 指定
	ShowCaptionAboveMedia bool             `json:"show_caption_above_media,omitempty"` // 可选。如果说明必须显示在消息媒体上方，则传递 True
	Width                 int              `json:"width,omitempty"`                    // 可选。视频宽度
	Height                int              `json:"height,omitempty"`                   // 可选。视频高度
	Duration              int              `json:"duration,omitempty"`                 // 可选。视频的持续时间（秒）
	SupportsStreaming     bool             `json:"supports_streaming,omitempty"`       // 可选。如果上传的视频适合流式传输，则传递 True
	HasSpoiler            bool             `json:"has_spoiler,omitempty"`              // 可选。如果视频需要用剧透动画遮盖，则传递 True
}

// InputMediaAnimation 表示要发送的动画文件（GIF 或无声的 H.264/MPEG-4 AVC 视频）。
type InputMediaAnimation struct {
	Type                  string           `json:"type"`                               // 结果的类型必须为动画
	Media                 RequestFileData  `json:"media"`                              // 要发送的文件。传递 file_id 以发送在 Telegram 服务器上存在的文件（推荐），传递 HTTP URL 让 Telegram 从互联网获取文件，或传递 “attach://<file_attach_name>” 以使用 multipart/form-data 上传新文件。
	Thumbnail             RequestFileData  `json:"thumb,omitempty"`                    // 可选。发送文件的缩略图；如果服务器支持文件的缩略图生成可以忽略。缩略图应为 JPEG 格式且大小小于 200 kB。缩略图的宽度和高度不得超过 320。如果文件不是通过 multipart/form-data 上传，则忽略。缩略图不能重用，只能作为新文件上传，因此可以传递 “attach://<file_attach_name>” 如果缩略图是通过 multipart/form-data 上传的。
	Caption               string           `json:"caption,omitempty"`                  // 可选。要发送的动画的说明，0-1024 个字符，经过实体解析
	ParseMode             string           `json:"parse_mode,omitempty"`               // 可选。动画说明中解析实体的模式。有关更多详细信息，请参见 [格式选项](https://core.telegram.org/bots/api#formatting-options)。
	CaptionEntities       []*MessageEntity `json:"caption_entities,omitempty"`         // 可选。出现在说明中的特殊实体列表，可以替代 parse_mode 指定
	ShowCaptionAboveMedia bool             `json:"show_caption_above_media,omitempty"` // 可选。如果说明必须显示在消息媒体上方，则传递 True
	Width                 int              `json:"width,omitempty"`                    // 可选。动画宽度
	Height                int              `json:"height,omitempty"`                   // 可选。动画高度
	Duration              int              `json:"duration,omitempty"`                 // 可选。动画的持续时间（秒）
	HasSpoiler            bool             `json:"has_spoiler,omitempty"`              // 可选。如果动画需要用剧透动画遮盖，则传递 True
}

// InputMediaAudio 表示要发送的音频文件（作为音乐处理）。
type InputMediaAudio struct {
	Type            string           `json:"type"`                       // 结果的类型必须为音频
	Media           RequestFileData  `json:"media"`                      // 要发送的文件。传递 file_id 以发送在 Telegram 服务器上存在的文件（推荐），传递 HTTP URL 让 Telegram 从互联网获取文件，或传递 “attach://<file_attach_name>” 以使用 multipart/form-data 上传新文件。
	Thumbnail       RequestFileData  `json:"thumb,omitempty"`            // 可选。发送文件的缩略图；如果服务器支持文件的缩略图生成可以忽略。缩略图应为 JPEG 格式且大小小于 200 kB。缩略图的宽度和高度不得超过 320。如果文件不是通过 multipart/form-data 上传，则忽略。缩略图不能重用，只能作为新文件上传，因此可以传递 “attach://<file_attach_name>” 如果缩略图是通过 multipart/form-data 上传的。
	Caption         string           `json:"caption,omitempty"`          // 可选。要发送的音频的说明，0-1024 个字符，经过实体解析
	ParseMode       string           `json:"parse_mode,omitempty"`       // 可选。音频说明中解析实体的模式。有关更多详细信息，请参见 [格式选项](https://core.telegram.org/bots/api#formatting-options)。
	CaptionEntities []*MessageEntity `json:"caption_entities,omitempty"` // 可选。出现在说明中的特殊实体列表，可以替代 parse_mode 指定
	Duration        int              `json:"duration,omitempty"`         // 可选。音频的持续时间（秒）
	Performer       string           `json:"performer,omitempty"`        // 可选。音频的表演者
	Title           string           `json:"title,omitempty"`            // 可选。音频的标题
}

// InputMediaDocument 表示要发送的一般文件。
type InputMediaDocument struct {
	Type                        string           `json:"type"`                                     // 结果的类型必须为文档
	Media                       RequestFileData  `json:"media"`                                    // 要发送的文件。传递 file_id 以发送在 Telegram 服务器上存在的文件（推荐），传递 HTTP URL 让 Telegram 从互联网获取文件，或传递 “attach://<file_attach_name>” 以使用 multipart/form-data 上传新文件。
	Thumbnail                   RequestFileData  `json:"thumb,omitempty"`                          // 可选。发送文件的缩略图；如果服务器支持文件的缩略图生成可以忽略。缩略图应为 JPEG 格式且大小小于 200 kB。缩略图的宽度和高度不得超过 320。如果文件不是通过 multipart/form-data 上传，则忽略。缩略图不能重用，只能作为新文件上传，因此可以传递 “attach://<file_attach_name>” 如果缩略图是通过 multipart/form-data 上传的。
	Caption                     string           `json:"caption,omitempty"`                        // 可选。要发送的文档的说明，0-1024 个字符，经过实体解析
	ParseMode                   string           `json:"parse_mode,omitempty"`                     // 可选。文档说明中解析实体的模式。有关更多详细信息，请参见 [格式选项](https://core.telegram.org/bots/api#formatting-options)。
	CaptionEntities             []*MessageEntity `json:"caption_entities,omitempty"`               // 可选。出现在说明中的特殊实体列表，可以替代 parse_mode 指定
	DisableContentTypeDetection bool             `json:"disable_content_type_detection,omitempty"` // 可选。对于通过 multipart/form-data 上传的文件，禁用自动服务器端内容类型检测。如果文档作为相册的一部分发送，总是为 True。
}

// Sticker 表示一个贴纸。
type Sticker struct {
	FileID           string        `json:"file_id"`                     // 此文件的标识符，可以用于下载或重用文件
	FileUniqueID     string        `json:"file_unique_id"`              // 此文件的唯一标识符，应该在时间上和不同的机器人中保持一致。不能用于下载或重用文件。
	Type             string        `json:"type"`                        // 贴纸的类型，目前为 “regular”、“masks”、“custom_emoji” 之一。贴纸的类型独立于其格式，格式由 is_animated 和 is_video 字段确定。
	Width            int           `json:"width"`                       // 贴纸宽度
	Height           int           `json:"height"`                      // 贴纸高度
	IsAnimated       bool          `json:"is_animated"`                 // 如果贴纸是动画的，则为 True
	IsVideo          bool          `json:"is_video"`                    // 如果贴纸是视频贴纸，则为 True
	Thumbnail        *PhotoSize    `json:"thumb,omitempty"`             // 可选。贴纸的缩略图，格式为 .WEBP 或 .JPG
	Emoji            string        `json:"emoji,omitempty"`             // 可选。与贴纸相关的表情符号
	SetName          string        `json:"set_name,omitempty"`          // 可选。贴纸所属的贴纸集名称
	PremiumAnimation *File         `json:"premium_animation,omitempty"` // 可选。贴纸的高级动画（如果贴纸是高级的）
	MaskPosition     *MaskPosition `json:"mask_position,omitempty"`     // 可选。对于面具贴纸，面具应放置的位置
	CustomEmojiID    string        `json:"custom_emoji_id,omitempty"`   // 可选。对于自定义表情贴纸，自定义表情的唯一标识符
	NeedsRepainting  bool          `json:"needs_repainting,omitempty"`  // 可选。如果贴纸必须重新涂色以适应消息中的文本颜色、Telegram Premium徽章的颜色、聊天照片中的白色或其他适当的颜色，则为 True
	FileSize         int64         `json:"file_size,omitempty"`         // 可选。文件大小（字节）
}

// StickerSet 表示一个贴纸集。
type StickerSet struct {
	Name        string     `json:"name"`            // 贴纸集名称
	Title       string     `json:"title"`           // 贴纸集标题
	StickerType string     `json:"sticker_type"`    // 贴纸集中的贴纸类型，目前为 “regular”、“masks”、“custom_emoji” 之一
	Stickers    []Sticker  `json:"stickers"`        // 所有贴纸集贴纸的列表
	Thumbnail   *PhotoSize `json:"thumb,omitempty"` // 可选。贴纸集的缩略图，格式为 .WEBP、.TGS 或 .WEBM
}

// MaskPosition 描述面具在面部默认放置的位置。
type MaskPosition struct {
	Point  string  `json:"point"`   // 面具应放置的面部部分。可以是 “foreheads”、“eyes”、“mouth” 或 “chin” 之一。
	XShift float64 `json:"x_shift"` // 沿 X 轴的偏移量，以面具的宽度为单位，从左到右测量。例如，选择 -1.0 会将面具放置在默认面具位置的左侧。
	YShift float64 `json:"y_shift"` // 沿 Y 轴的偏移量，以面具的高度为单位，从上到下测量。例如，选择 1.0 会将面具放置在默认面具位置的下方。
	Scale  float64 `json:"scale"`   // 面具缩放系数。例如，2.0 表示双倍大小。
}

// InputSticker 描述要添加到贴纸集的贴纸。
type InputSticker struct {
	Sticker      RequestFileData `json:"sticker"`       // 添加的贴纸。传递 file_id 作为字符串以发送已经存在于 Telegram 服务器上的文件，传递 HTTP URL 作为字符串让 Telegram 从互联网获取文件，或使用 multipart/form-data 上传新文件。动画和视频贴纸不能通过 HTTP URL 上传。
	Format       string          `json:"format"`        // 添加的贴纸的格式，必须为 “static” 对于 .WEBP 或 .PNG 图像，“animated” 对于 .TGS 动画，“video” 对于 WEBM 视频
	EmojiList    []string        `json:"emoji_list"`    // 与贴纸相关的 1-20 个表情符号的列表
	MaskPosition *MaskPosition   `json:"mask_position"` // 可选。面具应放置在面部的位置。仅适用于 “mask” 贴纸。
	Keywords     []string        `json:"keywords"`      // 可选。贴纸的 0-20 个搜索关键词，总长度不得超过 64 个字符。仅适用于 “regular” 和 “custom_emoji” 贴纸。
}

// InlineQuery 表示传入的内联查询。当用户发送空查询时，您的机器人可以返回一些默认或热门结果。
type InlineQuery struct {
	ID       string    `json:"id"`                  // 此查询的唯一标识符
	From     User      `json:"from"`                // 发送者
	Query    string    `json:"query"`               // 查询的文本（最多 256 个字符）
	Offset   string    `json:"offset"`              // 机器人可以控制返回结果的偏移量。
	ChatType string    `json:"chat_type,omitempty"` // 可选。发送内联查询的聊天类型。可以是 “sender” 代表与内联查询发送者的私人聊天，“private”、“group”、“supergroup” 或 “channel”。聊天类型在来自官方客户端和大多数第三方客户端的请求中应该始终已知，除非请求来自秘密聊天。
	Location *Location `json:"location,omitempty"`  // 可选。发送者位置，仅适用于请求用户位置的机器人
}

// InlineQueryResultsButton 表示在内联查询结果上方显示的按钮。您必须使用正好一个可选字段。
type InlineQueryResultsButton struct {
	Text           string      `json:"text"`                      // 按钮上的标签文本
	WebApp         *WebAppInfo `json:"web_app,omitempty"`         // 可选。用户按下按钮时将启动的 Web 应用的描述。Web 应用将能够使用 web_app_switch_inline_query 方法切换回内联模式。
	StartParameter *string     `json:"start_parameter,omitempty"` // 可选。用户按下按钮时发送给机器人的 /start 消息的深度链接参数。1-64 个字符，仅允许 A-Z、a-z、0-9、_ 和 -。例如，发送 YouTube 视频的内联机器人可以要求用户连接机器人到他们的 YouTube 帐号，以便相应地调整搜索结果。为此，它在结果上方显示一个“连接您的 YouTube 帐号”按钮，或者在显示任何内容之前显示。一旦完成，机器人可以提供一个 switch_inline 按钮，以便用户轻松返回到他们想要使用机器人内联功能的聊天中。
}

// InlineQueryResult 表示内联查询的一个结果。Telegram 客户端当前支持以下 20 种类型的结果
// 注意：所有在内联查询结果中传递的 URL 将对最终用户可用，因此必须假定它们是公开的。
type InlineQueryResult struct {
	InlineQueryResultArticle        InlineQueryResultArticle
	InlineQueryResultPhoto          InlineQueryResultPhoto
	InlineQueryResultGif            InlineQueryResultGIF
	InlineQueryResultMpeg4Gif       InlineQueryResultMPEG4GIF
	InlineQueryResultVideo          InlineQueryResultVideo
	InlineQueryResultAudio          InlineQueryResultAudio
	InlineQueryResultVoice          InlineQueryResultVoice
	InlineQueryResultDocument       InlineQueryResultDocument
	InlineQueryResultLocation       InlineQueryResultLocation
	InlineQueryResultVenue          InlineQueryResultVenue
	InlineQueryResultContact        InlineQueryResultContact
	InlineQueryResultGame           InlineQueryResultGame
	InlineQueryResultCachedPhoto    InlineQueryResultCachedPhoto
	InlineQueryResultCachedGif      InlineQueryResultCachedGIF
	InlineQueryResultCachedMpeg4Gif InlineQueryResultCachedMPEG4GIF
	InlineQueryResultCachedSticker  InlineQueryResultCachedSticker
	InlineQueryResultCachedDocument InlineQueryResultCachedDocument
	InlineQueryResultCachedVideo    InlineQueryResultCachedVideo
	InlineQueryResultCachedVoice    InlineQueryResultCachedVoice
	InlineQueryResultCachedAudio    InlineQueryResultCachedAudio
}

// InlineQueryResultArticle 表示一个指向文章或网页的链接。
type InlineQueryResultArticle struct {
	Type                string                `json:"type"`                   // 结果的类型必须是 article
	ID                  string                `json:"id"`                     // 该结果的唯一标识符，1-64 字节
	Title               string                `json:"title"`                  // 结果的标题
	InputMessageContent any                   `json:"input_message_content"`  // 要发送的消息内容
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty"` // 可选。附加到消息的内联键盘
	URL                 string                `json:"url,omitempty"`          // 可选。结果的 URL
	HideURL             bool                  `json:"hide_url,omitempty"`     // 可选。如果不想在消息中显示 URL，请传递 True
	Description         string                `json:"description,omitempty"`  // 可选。结果的简短描述
	ThumbnailURL        string                `json:"thumb_url,omitempty"`    // 可选。结果的缩略图 URL
	ThumbnailWidth      int                   `json:"thumb_width,omitempty"`  // 可选。缩略图宽度
	ThumbnailHeight     int                   `json:"thumb_height,omitempty"` // 可选。缩略图高度
}

// InlineQueryResultPhoto 表示一个指向照片的链接。
// 默认情况下，此照片将由用户发送，附带可选的标题。
// 或者，您可以使用 input_message_content 发送带有指定内容的消息，而不是照片。
type InlineQueryResultPhoto struct {
	Type                  string                `json:"type"`                               // 结果的类型必须是 photo
	ID                    string                `json:"id"`                                 // 该结果的唯一标识符，1-64 字节
	URL                   string                `json:"photo_url"`                          // 照片的有效 URL。照片必须为 JPEG 格式。照片大小不得超过 5MB
	ThumbnailURL          string                `json:"thumb_url"`                          // 照片的缩略图 URL
	Width                 int                   `json:"photo_width,omitempty"`              // 可选。照片的宽度
	Height                int                   `json:"photo_height,omitempty"`             // 可选。照片的高度
	Title                 string                `json:"title"`                              // 结果的标题
	Description           string                `json:"description,omitempty"`              // 可选。结果的简短描述
	Caption               string                `json:"caption,omitempty"`                  // 可选。要发送的照片的标题，0-1024 个字符在实体解析后
	ParseMode             string                `json:"parse_mode,omitempty"`               // 可选。解析标题中的实体的模式。有关更多细节，请参见 [格式选项](https://core.telegram.org/bots/api#formatting-options)
	CaptionEntities       []*MessageEntity      `json:"caption_entities,omitempty"`         // 可选。出现在标题中的特殊实体列表，可以代替 parse_mode 指定
	ShowCaptionAboveMedia bool                  `json:"show_caption_above_media,omitempty"` // 可选。如果标题必须显示在消息媒体上方，请传递 True
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`             // 可选。附加到消息的内联键盘
	InputMessageContent   any                   `json:"input_message_content,omitempty"`    // 可选。要发送的消息内容（而不是照片）
}

// InlineQueryResultGIF 表示一个指向动画 GIF 文件的链接。
// 默认情况下，此动画 GIF 文件将由用户发送，附带可选的标题。
// 或者，您可以使用 input_message_content 发送带有指定内容的消息，而不是动画。
type InlineQueryResultGIF struct {
	Type                  string                `json:"type"`                               // 结果的类型必须是 gif
	ID                    string                `json:"id"`                                 // 该结果的唯一标识符，1-64 字节
	URL                   string                `json:"gif_url"`                            // 动画 GIF 文件的有效 URL。文件大小不得超过 1MB
	Width                 int                   `json:"gif_width,omitempty"`                // 可选。GIF 的宽度
	Height                int                   `json:"gif_height,omitempty"`               // 可选。GIF 的高度
	Duration              int                   `json:"gif_duration,omitempty"`             // 可选。GIF 的持续时间（以秒为单位）
	ThumbnailURL          string                `json:"thumb_url"`                          // 结果的静态（JPEG 或 GIF）或动画（MPEG4）缩略图的 URL
	ThumbnailMimeType     string                `json:"thumb_mime_type,omitempty"`          // 可选。缩略图的 MIME 类型必须是“image/jpeg”、“image/gif”或“video/mp4”。默认为“image/jpeg”
	Title                 string                `json:"title,omitempty"`                    // 可选。结果的标题
	Caption               string                `json:"caption,omitempty"`                  // 可选。要发送的 GIF 文件的标题，0-1024 个字符在实体解析后
	ParseMode             string                `json:"parse_mode,omitempty"`               // 可选。解析标题中的实体的模式。有关更多细节，请参见 [格式选项](https://core.telegram.org/bots/api#formatting-options)
	CaptionEntities       []*MessageEntity      `json:"caption_entities,omitempty"`         // 可选。出现在标题中的特殊实体列表，可以代替 parse_mode 指定
	ShowCaptionAboveMedia bool                  `json:"show_caption_above_media,omitempty"` // 可选。如果标题必须显示在消息媒体上方，请传递 True
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`             // 可选。附加到消息的内联键盘
	InputMessageContent   any                   `json:"input_message_content,omitempty"`    // 可选。要发送的消息内容（而不是 GIF 动画）
}

// InlineQueryResultMPEG4GIF 表示一个指向视频动画（H.264/MPEG-4 AVC 视频，无声音）的链接。
// 默认情况下，此动画 MPEG-4 文件将由用户发送，附带可选的标题。
// 或者，您可以使用 input_message_content 发送带有指定内容的消息，而不是动画。
type InlineQueryResultMPEG4GIF struct {
	Type                  string                `json:"type"`                               // 结果的类型必须是 mpeg4_gif
	ID                    string                `json:"id"`                                 // 该结果的唯一标识符，1-64 字节
	URL                   string                `json:"mpeg4_url"`                          // MPEG4 文件的有效 URL。文件大小不得超过 1MB
	Width                 int                   `json:"mpeg4_width,omitempty"`              // 可选。视频宽度
	Height                int                   `json:"mpeg4_height,omitempty"`             // 可选。视频高度
	Duration              int                   `json:"mpeg4_duration,omitempty"`           // 可选。视频持续时间（以秒为单位）
	ThumbnailURL          string                `json:"thumb_url"`                          // 结果的静态（JPEG 或 GIF）或动画（MPEG4）缩略图的 URL
	ThumbnailMimeType     string                `json:"thumb_mime_type,omitempty"`          // 可选。缩略图的 MIME 类型必须是“image/jpeg”、“image/gif”或“video/mp4”。默认为“image/jpeg”
	Title                 string                `json:"title,omitempty"`                    // 可选。结果的标题
	Caption               string                `json:"caption,omitempty"`                  // 可选。要发送的 MPEG-4 文件的标题，0-1024 个字符在实体解析后
	ParseMode             string                `json:"parse_mode,omitempty"`               // 可选。解析标题中的实体的模式。有关更多细节，请参见 [格式选项](https://core.telegram.org/bots/api#formatting-options)
	CaptionEntities       []*MessageEntity      `json:"caption_entities,omitempty"`         // 可选。出现在标题中的特殊实体列表，可以代替 parse_mode 指定
	ShowCaptionAboveMedia bool                  `json:"show_caption_above_media,omitempty"` // 可选。如果标题必须显示在消息媒体上方，请传递 True
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`             // 可选。附加到消息的内联键盘
	InputMessageContent   any                   `json:"input_message_content,omitempty"`    // 可选。要发送的消息内容（而不是视频动画）
}

// InlineQueryResultVideo 表示一个指向包含嵌入视频播放器或视频文件的页面的链接。
// 默认情况下，此视频文件将由用户发送，附带可选的标题。
// 或者，您可以使用 input_message_content 发送带有指定内容的消息，而不是视频。
// 如果 InlineQueryResultVideo 消息包含嵌入视频（例如，YouTube），
// 您必须使用 input_message_content 替换其内容。
type InlineQueryResultVideo struct {
	Type                  string                `json:"type"`                               // 结果的类型必须是 video
	ID                    string                `json:"id"`                                 // 该结果的唯一标识符，1-64 字节
	URL                   string                `json:"video_url"`                          // 嵌入视频播放器或视频文件的有效 URL
	MimeType              string                `json:"mime_type"`                          // 视频 URL 内容的 MIME 类型，“text/html”或“video/mp4”
	ThumbnailURL          string                `json:"thumb_url"`                          // 视频的缩略图（仅 JPEG）的 URL
	Title                 string                `json:"title"`                              // 结果的标题
	Caption               string                `json:"caption,omitempty"`                  // 可选。要发送的视频的标题，0-1024 个字符在实体解析后
	ParseMode             string                `json:"parse_mode,omitempty"`               // 可选。解析视频标题中的实体的模式。有关更多细节，请参见 [格式选项](https://core.telegram.org/bots/api#formatting-options)
	CaptionEntities       []*MessageEntity      `json:"caption_entities,omitempty"`         // 可选。出现在标题中的特殊实体列表，可以代替 parse_mode 指定
	Width                 int                   `json:"video_width,omitempty"`              // 可选。视频宽度
	Height                int                   `json:"video_height,omitempty"`             // 可选。视频高度
	Duration              int                   `json:"video_duration,omitempty"`           // 可选。视频持续时间（以秒为单位）
	Description           string                `json:"description,omitempty"`              // 可选。结果的简短描述
	ShowCaptionAboveMedia bool                  `json:"show_caption_above_media,omitempty"` // 可选。如果标题必须显示在消息媒体上方，请传递 True
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`             // 可选。附加到消息的内联键盘
	InputMessageContent   any                   `json:"input_message_content,omitempty"`    // 可选。要发送的消息内容（而不是视频）。如果使用 InlineQueryResultVideo 发送 HTML 页面作为结果（例如，YouTube 视频），则此字段是必需的。
}

// InlineQueryResultAudio 表示一个指向 MP3 音频文件的链接。
// 默认情况下，此音频文件将由用户发送。
// 或者，您可以使用 input_message_content 发送带有指定内容的消息，而不是音频。
// 注意：这仅在 2016 年 4 月 9 日之后发布的 Telegram 版本中有效。
// 较旧的客户端将忽略它们。
type InlineQueryResultAudio struct {
	Type                string                `json:"type"`                            // 结果的类型必须是 audio
	ID                  string                `json:"id"`                              // 该结果的唯一标识符，1-64 字节
	URL                 string                `json:"audio_url"`                       // 音频文件的有效 URL
	Title               string                `json:"title"`                           // 标题
	Caption             string                `json:"caption,omitempty"`               // 可选。标题，0-1024 个字符在实体解析后
	ParseMode           string                `json:"parse_mode,omitempty"`            // 可选。解析音频标题中的实体的模式。有关更多细节，请参见 [格式选项](https://core.telegram.org/bots/api#formatting-options)
	CaptionEntities     []*MessageEntity      `json:"caption_entities,omitempty"`      // 可选。出现在标题中的特殊实体列表，可以代替 parse_mode 指定
	Performer           string                `json:"performer,omitempty"`             // 可选。表演者
	Duration            int                   `json:"audio_duration,omitempty"`        // 可选。音频持续时间（以秒为单位）
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty"`          // 可选。附加到消息的内联键盘
	InputMessageContent any                   `json:"input_message_content,omitempty"` // 可选。要发送的消息内容（而不是音频）
}

// InlineQueryResultVoice 表示一个指向 .OGG 容器中编码为 OPUS 的语音录音的链接。
// 默认情况下，此语音录音将由用户发送。
// 或者，您可以使用 input_message_content 发送带有指定内容的消息，而不是语音消息。
// 注意：这仅在 2016 年 4 月 9 日之后发布的 Telegram 版本中有效。
// 较旧的客户端将忽略它们。
type InlineQueryResultVoice struct {
	Type                string                `json:"type"`                            // 结果的类型必须是 voice
	ID                  string                `json:"id"`                              // 该结果的唯一标识符，1-64 字节
	URL                 string                `json:"voice_url"`                       // 语音录音的有效 URL
	Title               string                `json:"title"`                           // 录音标题
	Caption             string                `json:"caption,omitempty"`               // 可选。标题，0-1024 个字符在实体解析后
	ParseMode           string                `json:"parse_mode,omitempty"`            // 可选。解析语音消息标题中的实体的模式。有关更多细节，请参见 [格式选项](https://core.telegram.org/bots/api#formatting-options)
	CaptionEntities     []*MessageEntity      `json:"caption_entities,omitempty"`      // 可选。出现在标题中的特殊实体列表，可以代替 parse_mode 指定
	Duration            int                   `json:"voice_duration,omitempty"`        // 可选。录音持续时间（以秒为单位）
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty"`          // 可选。附加到消息的内联键盘
	InputMessageContent any                   `json:"input_message_content,omitempty"` // 可选。要发送的消息内容（而不是语音录音）
}

// InlineQueryResultDocument Represents a link to a file.
// By default, this file will be sent by the user with an optional caption.
// Alternatively, you can use input_message_content to send a message with the specified content instead of the file.
// Currently, only .PDF and .ZIP files can be sent using this method.
// Note: This will only work in Telegram versions released after 9 April 2016.
// Older clients will ignore them.
type InlineQueryResultDocument struct {
	Type                string                `json:"type"`                            // the type of the result must be documented
	ID                  string                `json:"id"`                              // Unique identifier for this result, 1-64 bytes
	Title               string                `json:"title"`                           // title for the result
	Caption             string                `json:"caption,omitempty"`               // Optional. Caption of the document to be sent, 0-1024 characters after entities parsing
	ParseMode           string                `json:"parse_mode,omitempty"`            // Optional. Mode for parsing entities in the document caption. See [formatting options](https://core.telegram.org/bots/api#formatting-options) for more details.
	CaptionEntities     []*MessageEntity      `json:"caption_entities,omitempty"`      // Optional. List of special entities that appear in the caption, which can be specified instead of parse_mode
	URL                 string                `json:"document_url"`                    // A valid URL for the file
	MimeType            string                `json:"mime_type"`                       // MIME type of the content of the file, either “application/pdf” or “application/zip”
	Description         string                `json:"description,omitempty"`           // Optional. Short description of the result
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty"`          // Optional. Inline keyboard attached to the message
	InputMessageContent any                   `json:"input_message_content,omitempty"` // Optional. Content of the message to be sent instead of the file
	ThumbnailURL        string                `json:"thumb_url,omitempty"`             // Optional. URL of the thumbnail (JPEG only) for the file
	ThumbnailWidth      int                   `json:"thumb_width,omitempty"`           // Optional. Thumbnail width
	ThumbnailHeight     int                   `json:"thumb_height,omitempty"`          // Optional. Thumbnail height
}

// InlineQueryResultLocation Represents a location on a map.
// By default, the location will be sent by the user.
// Alternatively,
// you can use input_message_content to send a message with the specified content instead of the location.
// Note: This will only work in Telegram versions released after 9 April 2016.
// Older clients will ignore them.
type InlineQueryResultLocation struct {
	Type                 string                `json:"type"`                             // type of the result must be location
	ID                   string                `json:"id"`                               // Unique identifier for this result, 1-64 Bytes
	Latitude             float64               `json:"latitude"`                         // Location latitude in degrees
	Longitude            float64               `json:"longitude"`                        // Location longitude in degrees
	Title                string                `json:"title"`                            // Location title
	HorizontalAccuracy   float64               `json:"horizontal_accuracy,omitempty"`    // Optional. The radius of uncertainty for the location, measured in meters; 0-1500
	LivePeriod           int                   `json:"live_period,omitempty"`            // Optional. The Period in seconds for which the location can be updated should be between 60 and 86400.
	Heading              int                   `json:"heading,omitempty"`                // Optional. For live locations, the direction in which the user is moving in degrees. It Must be between 1 and 360 if specified.
	ProximityAlertRadius int                   `json:"proximity_alert_radius,omitempty"` // Optional. For live locations, a maximum distance for proximity alerts about approaching another chat member, in meters. It Must be between 1 and 100000 if specified.
	ReplyMarkup          *InlineKeyboardMarkup `json:"reply_markup,omitempty"`           // Optional. Inline keyboard attached to the message
	InputMessageContent  any                   `json:"input_message_content,omitempty"`  // Optional. Content of the message to be sent instead of the location
	ThumbnailURL         string                `json:"thumb_url,omitempty"`              // Optional. Url of the thumbnail for the result
	ThumbnailWidth       int                   `json:"thumb_width,omitempty"`            // Optional. Thumbnail width
	ThumbnailHeight      int                   `json:"thumb_height,omitempty"`           // Optional. Thumbnail height
}

// InlineQueryResultVenue Represents a venue.
// By default, the venue will be sent by the user.
// Alternatively, you can use input_message_content to send a message with the specified content instead of the venue.
// Note: This will only work in Telegram versions released after 9 April 2016.
// Older clients will ignore them.
type InlineQueryResultVenue struct {
	Type                string                `json:"type"`                            // type of the result must be venue
	ID                  string                `json:"id"`                              // Unique identifier for this result, 1-64 Bytes
	Latitude            float64               `json:"latitude"`                        // latitude of the venue location in degrees
	Longitude           float64               `json:"longitude"`                       // longitude of the venue location in degrees
	Title               string                `json:"title"`                           // title of the venue
	Address             string                `json:"address"`                         // address of the venue
	FoursquareID        string                `json:"foursquare_id,omitempty"`         // Optional. Foursquare identifier of the venue if known
	FoursquareType      string                `json:"foursquare_type,omitempty"`       // Optional. Foursquare type of the venue, if known. (For example, “arts_entertainment/default,” “arts_entertainment/aquarium” or “food/ice cream.”)
	GooglePlaceID       string                `json:"google_place_id,omitempty"`       // Optional. Google Places identifier of the venue
	GooglePlaceType     string                `json:"google_place_type,omitempty"`     // Optional. Google Places a type of the venue. See [supported types](https://developers.google.com/places/web-service/supported_types).
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty"`          // Optional. Inline keyboard attached to the message
	InputMessageContent any                   `json:"input_message_content,omitempty"` // Optional. Content of the message to be sent instead of the venue
	ThumbnailURL        string                `json:"thumb_url,omitempty"`             // Optional. Url of the thumbnail for the result
	ThumbnailWidth      int                   `json:"thumb_width,omitempty"`           // Optional. Thumbnail width
	ThumbnailHeight     int                   `json:"thumb_height,omitempty"`          // Optional. Thumbnail height
}

// InlineQueryResultContact Represents a contact with a phone number.
// By default, this contact will be sent by the user.
// Alternatively, you can use input_message_content to send a message with the specified content instead of the contact.
// Note: This will only work in Telegram versions released after 9 April 2016.
// Older clients will ignore them.
type InlineQueryResultContact struct {
	Type                string                `json:"type"`                            // type of the result must be contact
	ID                  string                `json:"id"`                              // Unique identifier for this result, 1-64 Bytes
	PhoneNumber         string                `json:"phone_number"`                    // contact's phone number
	FirstName           string                `json:"first_name"`                      // contact's first name
	LastName            string                `json:"last_name,omitempty"`             // Optional. Contact's last name
	VCard               string                `json:"vcard,omitempty"`                 // Optional. Additional data about the contact in the form of a vCard, 0-2048 bytes
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty"`          // Optional. Inline keyboard attached to the message
	InputMessageContent any                   `json:"input_message_content,omitempty"` // Optional. Content of the message to be sent instead of the contact
	ThumbnailURL        string                `json:"thumb_url,omitempty"`             // Optional. Url of the thumbnail for the result
	ThumbnailWidth      int                   `json:"thumb_width,omitempty"`           // Optional. Thumbnail width
	ThumbnailHeight     int                   `json:"thumb_height,omitempty"`          // Optional. Thumbnail height
}

// InlineQueryResultGame Represents a Game.
// Note: This will only work in Telegram versions released after October 1, 2016.
// Older clients will not display any inline results if a game result is among them.
type InlineQueryResultGame struct {
	Type          string                `json:"type"`                   // type of the result must be game
	ID            string                `json:"id"`                     // Unique identifier for this result, 1-64 bytes
	GameShortName string                `json:"game_short_name"`        // Short name of the game
	ReplyMarkup   *InlineKeyboardMarkup `json:"reply_markup,omitempty"` // Optional. Inline keyboard attached to the message
}

// InlineQueryResultCachedPhoto Represents a link to a photo stored on the Telegram servers.
// By default, this photo will be sent by the user with an optional caption.
// Alternatively, you can use input_message_content to send a message with the specified content instead of the photo.
type InlineQueryResultCachedPhoto struct {
	Type                  string                `json:"type"`                               // type of the result must be photoed
	ID                    string                `json:"id"`                                 // Unique identifier for this result, 1-64 bytes
	PhotoID               string                `json:"photo_file_id"`                      // A valid file identifier of the photo
	Title                 string                `json:"title,omitempty"`                    // Optional. Title for the result
	Description           string                `json:"description,omitempty"`              // Optional. Short description of the result
	Caption               string                `json:"caption,omitempty"`                  // Optional. Caption of the photo to be sent, 0-1024 characters after entities parsing
	ParseMode             string                `json:"parse_mode,omitempty"`               // Optional. Mode for parsing entities in the photo caption. See [formatting options](https://core.telegram.org/bots/api#formatting-options) for more details.
	CaptionEntities       []*MessageEntity      `json:"caption_entities,omitempty"`         // Optional. List of special entities that appear in the caption, which can be specified instead of parse_mode
	ShowCaptionAboveMedia bool                  `json:"show_caption_above_media,omitempty"` // Optional. Pass True, if the caption must be shown above the message media
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`             // Optional. Inline keyboard attached to the message
	InputMessageContent   any                   `json:"input_message_content,omitempty"`    // Optional. Content of the message to be sent instead of the photo
}

// InlineQueryResultCachedGIF Represents a link to an animated GIF file stored on the Telegram servers.
// By default, this animated GIF file will be sent by the user with an optional caption.
// Alternatively, you can use input_message_content to send a message with specified content instead of the animation.
type InlineQueryResultCachedGIF struct {
	Type                  string                `json:"type"`                               // the type of the result must be gif
	ID                    string                `json:"id"`                                 // Unique identifier for this result, 1-64 bytes
	GifID                 string                `json:"gif_file_id"`                        // A valid file identifier for the GIF file
	Title                 string                `json:"title,omitempty"`                    // Optional. Title for the result
	Caption               string                `json:"caption,omitempty"`                  // Optional. Caption of the GIF file to be sent, 0-1024 characters after entities parsing
	ParseMode             string                `json:"parse_mode,omitempty"`               // Optional. Mode for parsing entities in the caption. See [formatting options](https://core.telegram.org/bots/api#formatting-options) for more details.
	CaptionEntities       []*MessageEntity      `json:"caption_entities,omitempty"`         // Optional. List of special entities that appear in the caption, which can be specified instead of parse_mode
	ShowCaptionAboveMedia bool                  `json:"show_caption_above_media,omitempty"` // Optional. Pass True, if the caption must be shown above the message media
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`             // Optional. Inline keyboard attached to the message
	InputMessageContent   any                   `json:"input_message_content,omitempty"`    // Optional. Content of the message to be sent instead of the GIF animation
}

// InlineQueryResultCachedMPEG4GIF Represents a link to a video animation
// (H.264/MPEG-4 AVC video without a sound) stored on the Telegram servers.
// By default, this animated MPEG-4 file will be sent by the user with an optional caption.
// Alternatively,
// you can use input_message_content to send a message with the specified content instead of the animation.
type InlineQueryResultCachedMPEG4GIF struct {
	Type                  string                `json:"type"`                               // type of the result must be mpeg4_gif
	ID                    string                `json:"id"`                                 // Unique identifier for this result, 1-64 bytes
	MPEG4FileID           string                `json:"mpeg4_file_id"`                      // A valid file identifier for the MPEG4 file
	Title                 string                `json:"title,omitempty"`                    // Optional. Title for the result
	Caption               string                `json:"caption,omitempty"`                  // Optional. Caption of the MPEG-4 file to be sent, 0-1024 characters after entities parsing
	ParseMode             string                `json:"parse_mode,omitempty"`               // Optional. Mode for parsing entities in the caption. See [formatting options](https://core.telegram.org/bots/api#formatting-options) for more details.
	CaptionEntities       []*MessageEntity      `json:"caption_entities,omitempty"`         // Optional. List of special entities that appear in the caption, which can be specified instead of parse_mode
	ShowCaptionAboveMedia bool                  `json:"show_caption_above_media,omitempty"` // Optional. Pass True, if the caption must be shown above the message media
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`             // Optional. Inline keyboard attached to the message
	InputMessageContent   any                   `json:"input_message_content,omitempty"`    // Optional. Content of the message to be sent instead of the video animation
}

// InlineQueryResultCachedSticker Represents a link to a sticker stored on the Telegram servers.
// By default, this sticker will be sent by the user.
// Alternatively, you can use input_message_content to send a message with the specified content instead of the sticker.
// Note:
// This will only work in Telegram versions
// released after 9 April 2016 for static stickers and after 06 July 2019 for animated stickers.
// Older clients will ignore them.
type InlineQueryResultCachedSticker struct {
	Type                string                `json:"type"`                            // the type of the result must be stickered
	ID                  string                `json:"id"`                              // Unique identifier for this result, 1-64 bytes
	StickerID           string                `json:"sticker_file_id"`                 // A valid file identifier of the sticker
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty"`          // Optional. Inline keyboard attached to the message
	InputMessageContent any                   `json:"input_message_content,omitempty"` // Optional. Content of the message to be sent instead of the sticker
}

// InlineQueryResultCachedDocument Represents a link to a file stored on the Telegram servers.
// By default, this file will be sent by the user with an optional caption.
// Alternatively, you can use input_message_content to send a message with the specified content instead of the file.
// Note: This will only work in Telegram versions released after 9 April 2016.
// Older clients will ignore them.
type InlineQueryResultCachedDocument struct {
	Type                string                `json:"type"`                            // the type of the result must be documented
	ID                  string                `json:"id"`                              // Unique identifier for this result, 1-64 bytes
	Title               string                `json:"title"`                           // title for the result
	DocumentID          string                `json:"document_file_id"`                // A valid file identifier for the file
	Description         string                `json:"description,omitempty"`           // Optional. Short description of the result
	Caption             string                `json:"caption,omitempty"`               // Optional. Caption of the document to be sent, 0-1024 characters after entities parsing
	ParseMode           string                `json:"parse_mode,omitempty"`            // Optional. Mode for parsing entities in the document caption. See [formatting options](https://core.telegram.org/bots/api#formatting-options) for more details.
	CaptionEntities     []*MessageEntity      `json:"caption_entities,omitempty"`      // Optional. List of special entities that appear in the caption, which can be specified instead of parse_mode
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty"`          // Optional. Inline keyboard attached to the message
	InputMessageContent any                   `json:"input_message_content,omitempty"` // Optional. Content of the message to be sent instead of the file
}

// InlineQueryResultCachedVideo Represents a link to a video file stored on the Telegram servers.
// By default, this video file will be sent by the user with an optional caption.
// Alternatively, you can use input_message_content to send a message with the specified content instead of the video.
type InlineQueryResultCachedVideo struct {
	Type                  string                `json:"type"`                               // type of the result must be video
	ID                    string                `json:"id"`                                 // Unique identifier for this result, 1-64 bytes
	VideoID               string                `json:"video_file_id"`                      // A valid file identifier for the video file
	Title                 string                `json:"title"`                              // title for the result
	Description           string                `json:"description,omitempty"`              // Optional. Short description of the result
	Caption               string                `json:"caption,omitempty"`                  // Optional. Caption of the video to be sent, 0-1024 characters after entities parsing
	ParseMode             string                `json:"parse_mode,omitempty"`               // Optional. Mode for parsing entities in the video caption. See [formatting options](https://core.telegram.org/bots/api#formatting-options) for more details.
	CaptionEntities       []*MessageEntity      `json:"caption_entities,omitempty"`         // Optional. List of special entities that appear in the caption, which can be specified instead of parse_mode
	ShowCaptionAboveMedia bool                  `json:"show_caption_above_media,omitempty"` // Optional. Pass True, if the caption must be shown above the message media
	ReplyMarkup           *InlineKeyboardMarkup `json:"reply_markup,omitempty"`             // Optional. Inline keyboard attached to the message
	InputMessageContent   any                   `json:"input_message_content,omitempty"`    // Optional. Content of the message to be sent instead of the video
}

// InlineQueryResultCachedVoice Represents a link to a voice message stored on the Telegram servers.
// By default, this voice message will be sent by the user.
// Alternatively,
// you can use input_message_content to send a message with the specified content instead of the voice message.
// Note: This will only work in Telegram versions released after 9 April 2016.
// Older clients will ignore them.
type InlineQueryResultCachedVoice struct {
	Type                string                `json:"type"`                            // the type of the result must be voice
	ID                  string                `json:"id"`                              // Unique identifier for this result, 1-64 bytes
	VoiceID             string                `json:"voice_file_id"`                   // A valid file identifier for the voice message
	Title               string                `json:"title"`                           // Voice message title
	Caption             string                `json:"caption,omitempty"`               // Optional. Caption, 0-1024 characters after entities parsing
	ParseMode           string                `json:"parse_mode,omitempty"`            // Optional. Mode for parsing entities in the voice message caption. See [formatting options](https://core.telegram.org/bots/api#formatting-options) for more details.
	CaptionEntities     []*MessageEntity      `json:"caption_entities,omitempty"`      // Optional. List of special entities that appear in the caption, which can be specified instead of parse_mode
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty"`          // Optional. Inline keyboard attached to the message
	InputMessageContent any                   `json:"input_message_content,omitempty"` // Optional. Content of the message to be sent instead of the voice message
}

// InlineQueryResultCachedAudio Represents a link to an MP3 audio file stored on the Telegram servers.
// By default, this audio file will be sent by the user.
// Alternatively, you can use input_message_content to send a message with the specified content instead of the audio.
// Note: This will only work in Telegram versions released after 9 April 2016.
// Older clients will ignore them.
type InlineQueryResultCachedAudio struct {
	Type                string                `json:"type"`                            // The Type of the result must be audio
	ID                  string                `json:"id"`                              // Unique identifier for this result, 1-64 bytes
	AudioID             string                `json:"audio_file_id"`                   // A valid file identifier for the audio file
	Caption             string                `json:"caption,omitempty"`               // Optional. Caption, 0-1024 characters after entities parsing
	ParseMode           string                `json:"parse_mode,omitempty"`            // Optional. Mode for parsing entities in the audio caption. See [formatting options](https://core.telegram.org/bots/api#formatting-options) for more details.
	CaptionEntities     []*MessageEntity      `json:"caption_entities,omitempty"`      // Optional. List of special entities that appear in the caption, which can be specified instead of parse_mode
	ReplyMarkup         *InlineKeyboardMarkup `json:"reply_markup,omitempty"`          // Optional. Inline keyboard attached to the message
	InputMessageContent any                   `json:"input_message_content,omitempty"` // Optional. Content of the message to be sent instead of the audio
}

// InputMessageContent Represents the content of a message to be sent as a result of an inline query.
// Telegram clients currently support the following five types:
type InputMessageContent struct {
	InputTextMessageContent     InputTextMessageContent
	InputLocationMessageContent InputLocationMessageContent
	InputVenueMessageContent    InputVenueMessageContent
	InputContactMessageContent  InputContactMessageContent
	InputInvoiceMessageContent  InputInvoiceMessageContent
}

type InputTextMessageContent struct {
	Text               string              `json:"message_text"`                   // text of the message to be sent, 1-4096 characters
	ParseMode          string              `json:"parse_mode,omitempty"`           // Optional. Mode for parsing entities in the message text. See formatting options for more details.
	Entities           []MessageEntity     `json:"entities,omitempty"`             // Optional. List of special entities that appear in message text, which can be specified instead of parse_mode
	LinkPreviewOptions *LinkPreviewOptions `json:"link_preview_options,omitempty"` // Optional. Link preview generation options for the message
}

// InputLocationMessageContent Represents the content of a location message to be sent as the result of an inline query.
type InputLocationMessageContent struct {
	Latitude             float64 `json:"latitude"`                         // latitude of the location in degrees
	Longitude            float64 `json:"longitude"`                        // longitude of the location in degrees
	HorizontalAccuracy   float64 `json:"horizontal_accuracy,omitempty"`    // Optional. The radius of uncertainty for the location, measured in meters; 0-1500
	LivePeriod           int     `json:"live_period,omitempty"`            // Optional. The Period in seconds for which the location can be updated should be between 60 and 86400.
	Heading              int     `json:"heading,omitempty"`                // Optional. For live locations, the direction in which the user is moving in degrees. It Must be between 1 and 360 if specified.
	ProximityAlertRadius int     `json:"proximity_alert_radius,omitempty"` // Optional. For live locations, a maximum distance for proximity alerts about approaching another chat member, in meters. It Must be between 1 and 100000 if specified.
}

// InputVenueMessageContent Represents the content of a venue message to be sent as the result of an inline query.
type InputVenueMessageContent struct {
	Latitude        float64 `json:"latitude"`                    // Latitude of the venue in degrees
	Longitude       float64 `json:"longitude"`                   // Longitude of the venue in degrees
	Title           string  `json:"title"`                       // Name of the venue
	Address         string  `json:"address"`                     // Address of the venue
	FoursquareID    string  `json:"foursquare_id,omitempty"`     // Optional. Foursquare identifier of the venue, if known
	FoursquareType  string  `json:"foursquare_type,omitempty"`   // Optional. Foursquare type of the venue, if known. (For example, “arts_entertainment/default,” “arts_entertainment/aquarium” or “food/ice cream.”)
	GooglePlaceID   string  `json:"google_place_id,omitempty"`   // Optional. Google Places identifier of the venue
	GooglePlaceType string  `json:"google_place_type,omitempty"` // Optional. Google Places a type of the venue. See [supported types](https://developers.google.com/places/web-service/supported_types).
}

// InputContactMessageContent Represents the content of a contact message to be sent as the result of an inline query.
type InputContactMessageContent struct {
	PhoneNumber string `json:"phone_number"`        // Contact's phone number
	FirstName   string `json:"first_name"`          // Contact's first name
	LastName    string `json:"last_name,omitempty"` // Optional. Contact's last name
	VCard       string `json:"vcard,omitempty"`     // Optional. Additional data about the contact in the form of a vCard, 0-2048 bytes
}

// InputInvoiceMessageContent Represents the content of an invoice message to be sent as the result of an inline query.
type InputInvoiceMessageContent struct {
	Title                     string         `json:"title"`                                   // Product name, 1-32 characters
	Description               string         `json:"description"`                             // Product description, 1-255 characters
	Payload                   string         `json:"payload"`                                 // Bot-defined invoice payload, 1-128 bytes. This will not be displayed to the user, use it for your internal processes.
	ProviderToken             string         `json:"provider_token"`                          // Payment provider token, obtained via @BotFather
	Currency                  string         `json:"currency"`                                // Three-letter ISO 4217 currency code
	Prices                    []LabeledPrice `json:"prices"`                                  // Price breakdown, a JSON-serialized list of components (e.g., product price, tax, discount, delivery cost, delivery tax, bonus, etc.)
	MaxTipAmount              int            `json:"max_tip_amount,omitempty"`                // Optional. The maximum accepted amount for tips in the smallest units of the currency (integer, not float/double). For example, for a maximum tip of US$ 1.45 pass max_tip_amount = 145. See the exp parameter in currencies.json, it shows the number of digits past the decimal point for each currency (2 for the majority of currencies). Default to 0
	SuggestedTipAmounts       []int          `json:"suggested_tip_amounts,omitempty"`         // Optional. A JSON-serialized array of suggested amounts of tip in the smallest units of the currency (integer, not float/double). At most four suggested tip amounts can be specified. The suggested tip amounts must be positive, passed in a strictly increased order and must not exceed max_tip_amount.
	ProviderData              string         `json:"provider_data,omitempty"`                 // Optional. A JSON-serialized object for data about the invoice, which will be shared with the payment provider. The payment provider should provide a detailed description of the required fields.
	PhotoURL                  string         `json:"photo_url,omitempty"`                     // Optional. URL of the product photo for the invoice. It Can be a photo of the goods or a marketing image for a service.
	PhotoSize                 int            `json:"photo_size,omitempty"`                    // Optional. Photo size in bytes
	PhotoWidth                int            `json:"photo_width,omitempty"`                   // Optional. Photo width
	PhotoHeight               int            `json:"photo_height,omitempty"`                  // Optional. Photo height
	NeedName                  bool           `json:"need_name,omitempty"`                     // Optional. Pass True, if you require the user's full name to complete the order
	NeedPhoneNumber           bool           `json:"need_phone_number,omitempty"`             // Optional. Pass True, if you require the user's phone number to complete the order
	NeedEmail                 bool           `json:"need_email,omitempty"`                    // Optional. Pass True, if you require the user's email address to complete the order
	NeedShippingAddress       bool           `json:"need_shipping_address,omitempty"`         // Optional. Pass True, if you require the user's shipping address to complete the order
	SendPhoneNumberToProvider bool           `json:"send_phone_number_to_provider,omitempty"` // Optional. Pass True, if the user's phone number should be sent to the provider
	SendEmailToProvider       bool           `json:"send_email_to_provider,omitempty"`        // Optional. Pass True, if the user's email address should be sent to the provider
	IsFlexible                bool           `json:"is_flexible,omitempty"`                   // Optional. Pass True, if the final price depends on the shipping method
}

// ChosenInlineResult Represents a result of an inline query that was chosen by the user and sent to their chat partner.
// Note: It is necessary to enable inline feedback via @BotFather in order to receive these objects in updates.
type ChosenInlineResult struct {
	ResultID        string    `json:"result_id"`                   // The unique identifier for the result that was chosen
	From            User      `json:"from"`                        // The user that chose the result
	Location        *Location `json:"location,omitempty"`          // Optional. Sender location, only for bots that require user location
	InlineMessageID string    `json:"inline_message_id,omitempty"` // Optional. Identifier of the sent inline message. Available only if there is an inline keyboard attached to the message. It Will be also received in callback queries and can be used to edit the message.
	Query           string    `json:"query"`                       // The query that was used to obtain the result
}

// SentWebAppMessage Describes an inline message sent by a Web App on behalf of a user.
type SentWebAppMessage struct {
	InlineMessageID string `json:"inline_message_id"` // Optional. Identifier of the sent inline message. Available only if there is an inline keyboard attached to the message.
}

// LabeledPrice Represents a portion of the price for goods or services.
type LabeledPrice struct {
	Label  string `json:"label"`  // Portion label
	Amount int    `json:"amount"` // Price of the product in the smallest units of the currency (integer, not float/double). For example, for a price of US$ 1.45 pass amount = 145. See the exp parameter in currencies.json, it shows the number of digits past the decimal point for each currency (2 for the majority of currencies).
}

// Invoice Contains basic information about an invoice
type Invoice struct {
	Title          string `json:"title"`           // Product name
	Description    string `json:"description"`     // Product description
	StartParameter string `json:"start_parameter"` // Unique bot deep-linking parameter that can be used to generate this invoice
	Currency       string `json:"currency"`        // Three-letter ISO 4217 [currency](https://core.telegram.org/bots/payments#supported-currencies) code
	TotalAmount    int    `json:"total_amount"`    // Total price in the smallest units of the currency (integer, not float/double). For example, for a price of US$ 1.45 pass amount = 145. See the exp parameter in [currencies.json](https://core.telegram.org/bots/payments/currencies.json), it shows the number of digits past the decimal point for each currency (2 for the majority of currencies).
}

// ShippingAddress Represents a shipping address.
type ShippingAddress struct {
	CountryCode string `json:"country_code"` // Two-letter ISO 3166-1 alpha-2 country code
	State       string `json:"state"`        // State, if applicable
	City        string `json:"city"`         // City
	StreetLine1 string `json:"street_line1"` // First line for the address
	StreetLine2 string `json:"street_line2"` // Second line for the address
	PostCode    string `json:"post_code"`    // Address post code
}

// OrderInfo Represents information about an order.
type OrderInfo struct {
	Name            string           `json:"name,omitempty"`             // Optional. Username
	PhoneNumber     string           `json:"phone_number,omitempty"`     // Optional. User's phone number
	Email           string           `json:"email,omitempty"`            // Optional. User email
	ShippingAddress *ShippingAddress `json:"shipping_address,omitempty"` // Optional. User shipping address
}

// ShippingOption Represents one shipping option.
type ShippingOption struct {
	ID     string         `json:"id"`     // Shipping option identifier
	Title  string         `json:"title"`  // Option title
	Prices []LabeledPrice `json:"prices"` // List of price portions
}

// SuccessfulPayment Contains basic information about a successful payment.
type SuccessfulPayment struct {
	Currency                string     `json:"currency"`                     // Three-letter ISO 4217 currency code
	TotalAmount             int        `json:"total_amount"`                 // Total price in the smallest units of the currency (integer, not float/double). For example, for a price of US$ 1.45 pass amount = 145. See the exp parameter in currencies.json, it shows the number of digits past the decimal point for each currency (2 for the majority of currencies).
	InvoicePayload          string     `json:"invoice_payload"`              // Bot specified invoice payload
	ShippingOptionID        string     `json:"shipping_option_id,omitempty"` // Optional. Identifier of the shipping option chosen by the user
	OrderInfo               *OrderInfo `json:"order_info,omitempty"`         // Optional. Order information provided by the user
	TelegramPaymentChargeID string     `json:"telegram_payment_charge_id"`   // Telegram payment identifier
	ProviderPaymentChargeID string     `json:"provider_payment_charge_id"`   // Provider payment identifier
}

// ShippingQuery Contains information about an incoming shipping query.
type ShippingQuery struct {
	ID              string          `json:"id"`               // Unique query identifier
	From            User            `json:"from"`             // User who sent the query
	InvoicePayload  string          `json:"invoice_payload"`  // Bot specified invoice payload
	ShippingAddress ShippingAddress `json:"shipping_address"` // User specified shipping address
}

// PreCheckoutQuery Contains information about an incoming pre-checkout query.
type PreCheckoutQuery struct {
	ID               string     `json:"id"`                           // Unique query identifier
	From             User       `json:"from"`                         // User who sent the query
	Currency         string     `json:"currency"`                     // Three-letter ISO 4217 currency code
	TotalAmount      int        `json:"total_amount"`                 // Total price in the smallest units of the currency (integer, not float/double). For example, for a price of US$ 1.45 pass amount = 145. See the exp parameter in currencies.json, it shows the number of digits past the decimal point for each currency (2 for the majority of currencies).
	InvoicePayload   string     `json:"invoice_payload"`              // Bot specified invoice payload
	ShippingOptionID string     `json:"shipping_option_id,omitempty"` // Optional. Identifier of the shipping option chosen by the user
	OrderInfo        *OrderInfo `json:"order_info,omitempty"`         // Optional. Order information provided by the user
}

// PassportData Describes Telegram Passport data shared with the bot by the user.
type PassportData struct {
	Data        []EncryptedPassportElement `json:"data"`        // Array with information about documents and other Telegram Passport elements that was shared with the bot
	Credentials EncryptedCredentials       `json:"credentials"` // Encrypted credentials required to decrypt the data
}

// PassportFile Represents a file uploaded to Telegram Passport.
// Currently, all Telegram Passport files are in JPEG format when decrypted and don't exceed 10MB.
type PassportFile struct {
	FileID       string `json:"file_id"`        // Identifier for this file, which can be used to download or reuse the file
	FileUniqueID string `json:"file_unique_id"` // Unique identifier for this file, which is supposed to be the same over time and for different bots. Can't be used to download or reuse the file.
	FileSize     int    `json:"file_size"`      // File size in bytes
	FileDate     int64  `json:"file_date"`      // Unix time when the file was uploaded
}

// EncryptedPassportElement Describes documents or other Telegram Passport elements shared with the bot by the user.
type EncryptedPassportElement struct {
	Type        string          `json:"type"`                   // Element type. One of “personal_details”, “passport”, “driver_license”, “identity_card”, “internal_passport”, “address”, “utility_bill”, “bank_statement”, “rental_agreement”, “passport_registration”, “temporary_registration”, “phone_number”, “email”.
	Data        string          `json:"data,omitempty"`         // Optional. Base64-encoded encrypted Telegram Passport element data provided by the user, available for “personal_details,” “passport,” “driver_license,” “identity_card,” “internal_passport” and “address” types. Can be decrypted and verified using the accompanying [EncryptedCredentials](https://core.telegram.org/bots/api#encryptedcredentials).
	PhoneNumber string          `json:"phone_number,omitempty"` // Optional. User's verified phone number, available only for “phone_number” type
	Email       string          `json:"email,omitempty"`        // Optional. User's verified email address, available only for “email” type
	Files       []*PassportFile `json:"files,omitempty"`        // Optional. Array of encrypted files with documents provided by the user, available for “utility_bill,” “bank_statement,” “rental_agreement,” “passport_registration” and “temporary_registration” types. Files can be decrypted and verified using the accompanying [EncryptedCredentials](https://core.telegram.org/bots/api#encryptedcredentials).
	FrontSide   *PassportFile   `json:"front_side,omitempty"`   // Optional. Encrypted file with the front side of the document, provided by the user. Available for “passport,” “driver_license,” “identity_card” and “internal_passport.” The file can be decrypted and verified using the accompanying [EncryptedCredentials](https://core.telegram.org/bots/api#encryptedcredentials).
	ReverseSide *PassportFile   `json:"reverse_side,omitempty"` // Optional. Encrypted file with the reverse side of the document, provided by the user. Available for “driver_license” and “identity_card.” The file can be decrypted and verified using the accompanying [EncryptedCredentials](https://core.telegram.org/bots/api#encryptedcredentials).
	Selfie      *PassportFile   `json:"selfie,omitempty"`       // Optional. Encrypted file with the selfie of the user holding a document, provided by the user; available for “passport,” “driver_license,” “identity_card” and “internal_passport.” The file can be decrypted and verified using the accompanying [EncryptedCredentials](https://core.telegram.org/bots/api#encryptedcredentials).
	Translation []*PassportFile `json:"translation,omitempty"`  // Optional. Array of encrypted files with translated versions of documents provided by the user. Available if requested for “passport,” “driver_license,” “identity_card,” “internal_passport,” “utility_bill,” “bank_statement,” “rental_agreement,” “passport_registration” and “temporary_registration” types. Files can be decrypted and verified using the accompanying [EncryptedCredentials](https://core.telegram.org/bots/api#encryptedcredentials).
	Hash        string          `json:"hash"`                   // Base64-encoded element hash for using in [PassportElementErrorUnspecified](https://core.telegram.org/bots/api#passportelementerrorunspecified)
}

// EncryptedCredentials Describes data required for decrypting and authenticating EncryptedPassportElement.
// See the Telegram Passport Documentation for a complete description of the data decryption and authentication processes.
type EncryptedCredentials struct {
	Data   string `json:"data"`   // Base64-encoded encrypted JSON-serialized data with unique user's payload, data hashes and secrets required for [EncryptedPassportElement](https://core.telegram.org/bots/api#encryptedpassportelement) decryption and authentication
	Hash   string `json:"hash"`   // Base64-encoded data hash for data authentication
	Secret string `json:"secret"` // Base64-encoded secret, encrypted with the bot's public RSA key, required for data decryption
}

// PassportElementError Represents an error in the Telegram Passport element which was submitted that should be resolved by the user.
// It should be one of:
type PassportElementError struct {
	PassportElementErrorDataField        PassportElementErrorDataField
	PassportElementErrorFrontSide        PassportElementErrorFrontSide
	PassportElementErrorReverseSide      PassportElementErrorReverseSide
	PassportElementErrorSelfie           PassportElementErrorSelfie
	PassportElementErrorFile             PassportElementErrorFile
	PassportElementErrorFiles            PassportElementErrorFiles
	PassportElementErrorTranslationFile  PassportElementErrorTranslationFile
	PassportElementErrorTranslationFiles PassportElementErrorTranslationFiles
	PassportElementErrorUnspecified      PassportElementErrorUnspecified
}

// PassportElementErrorDataField Represents an issue in one of the data fields that was provided by the user.
// The error is considered resolved when the field's value changes.
type PassportElementErrorDataField struct {
	Source    string `json:"source"`     // Error source, must be data
	Type      string `json:"type"`       // The section of the user's Telegram Passport, which has the error, one of “personal_details,” “passport,” “driver_license,” “identity_card,” “internal_passport,” “address”
	FieldName string `json:"field_name"` // Name of the data field which has the error
	DataHash  string `json:"data_hash"`  // Base64-encoded data hash
	Message   string `json:"message"`    // Error message
}

// PassportElementErrorFrontSide Represents an issue with the front side of a document.
// The error is considered resolved when the file with the front side of the document changes.
type PassportElementErrorFrontSide struct {
	Source   string `json:"source"`    // Error source must be front_side
	Type     string `json:"type"`      // The section of the user's Telegram Passport which has the issue, one of “passports,” “driver_license,” “identity_card,” “internal_passport”
	FileHash string `json:"file_hash"` // Base64-encoded hash of the file with the front side of the document
	Message  string `json:"message"`   // Error message
}

// PassportElementErrorReverseSide Represents an issue with the reverse side of a document.
// The error is considered resolved when the file with the reverse side of the document changes.
type PassportElementErrorReverseSide struct {
	Source   string `json:"source"`    // Error source, must be reverse_side
	Type     string `json:"type"`      // The section of the user's Telegram Passport which has the issue, one of “driver_license,” “identity_card”
	FileHash string `json:"file_hash"` // Base64-encoded hash of the file with the reverse side of the document
	Message  string `json:"message"`   // Error message
}

// PassportElementErrorSelfie Represents an issue with the selfie with a document.
// The error is considered resolved when the file with the selfie changes.
type PassportElementErrorSelfie struct {
	Source   string `json:"source"`    // Error source, must be selfie
	Type     string `json:"type"`      // The section of the user's Telegram Passport which has the issue, one of “passports,” “driver_license,” “identity_card,” “internal_passport”
	FileHash string `json:"file_hash"` // Base64-encoded hash of the file with the selfie
	Message  string `json:"message"`   // Error message
}

// PassportElementErrorFile Represents an issue with a document scan.
// The error is considered resolved when the file with the document scan changes.
type PassportElementErrorFile struct {
	Source   string `json:"source"`    // Error source must be filed
	Type     string `json:"type"`      // The section of the user's Telegram Passport which has the issue, one of “utility_bill,” “bank_statement,” “rental_agreement,” “passport_registration,” “temporary_registration”
	FileHash string `json:"file_hash"` // Base64-encoded file hash
	Message  string `json:"message"`   // Error message
}

// PassportElementErrorFiles Represents an issue with a list of scans.
// The error is considered resolved when the list of files containing the scan changes.
type PassportElementErrorFiles struct {
	Source     string   `json:"source"`      // Error source, must be files
	Type       string   `json:"type"`        // The section of the user's Telegram Passport which has the issue, one of “utility_bill,” “bank_statement,” “rental_agreement,” “passport_registration,” “temporary_registration”
	FileHashes []string `json:"file_hashes"` // List of base64-encoded file hashes
	Message    string   `json:"message"`     // Error message
}

// PassportElementErrorTranslationFile Represents an issue with one of the files
// that constitute the translation of a document.
// The error is considered resolved when the file changes.
type PassportElementErrorTranslationFile struct {
	Source   string   `json:"source"`    // Error source, must be translation_file
	Type     string   `json:"type"`      // Type of element of the user's Telegram Passport which has the issue, one of “passports,” “driver_license,” “identity_card,” “internal_passport,” “utility_bill,” “bank_statement,” “rental_agreement,” “passport_registration,” “temporary_registration”
	FileHash []string `json:"file_hash"` // Base64-encoded file hash
	Message  string   `json:"message"`   // Error message
}

// PassportElementErrorTranslationFiles Represents an issue with the translated version of a document.
// The error is considered resolved when a file with the document translation changes.
type PassportElementErrorTranslationFiles struct {
	Source     string   `json:"source"`      // Error source, must be translation_files
	Type       string   `json:"type"`        // Type of element of the user's Telegram Passport which has the issue, one of “passports,” “driver_license,” “identity_card,” “internal_passport,” “utility_bill,” “bank_statement,” “rental_agreement,” “passport_registration,” “temporary_registration”
	FileHashes []string `json:"file_hashes"` // List of base64-encoded file hashes
	Message    string   `json:"message"`     // Error message
}

// PassportElementErrorUnspecified Represents an issue in an unspecified place.
// The error is considered resolved when new data is added.
type PassportElementErrorUnspecified struct {
	Source   string `json:"source"`    // Error source must be unspecified
	Type     string `json:"type"`      // Type of element of the user's Telegram Passport which has the issue
	FileHash string `json:"file_hash"` // Base64-encoded element hash
	Message  string `json:"message"`   // Error message
}

// Game Represents a game. Use BotFather to create and edit games, their short names will act as unique identifiers.
type Game struct {
	Title        string           `json:"title"`                   // Title of the game
	Description  string           `json:"description"`             // Description of the game
	Photo        []PhotoSize      `json:"photo"`                   // Photo that will be displayed in the game message in chats.
	Text         string           `json:"text,omitempty"`          // Optional. Brief description of the game or high scores included in the game message. It Can be automatically edited to include current high scores for the game when the bot calls setGameScore, or manually edited using editMessageText. 0-4096 characters.
	TextEntities []*MessageEntity `json:"text_entities,omitempty"` // Optional. Special entities that appear in text, such as usernames, URLs, bot commands, etc.
	Animation    *Animation       `json:"animation,omitempty"`     // Optional. Animation that will be displayed in the game message in chats. Upload via BotFather
}

// CallbackGame A placeholder currently holds no information. Use BotFather to set up your game.
type CallbackGame struct{}

// GameHighScore Represents one row of the high scores table for a game
type GameHighScore struct {
	Position int  `json:"position"` // Position in high-score table for the game
	User     User `json:"user"`     // User
	Score    int  `json:"score"`    // Score
}
