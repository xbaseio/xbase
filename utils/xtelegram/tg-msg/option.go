package tgmsg

import (
	//tgbutton "github.com/xbaseio/xbase/utils/xtelegram/tg-button"
	tgtypes "github.com/xbaseio/xbase/utils/xtelegram/tg-types"
)

type Option func(o *telegramMessage)

// 按钮绑定参数密码
func WithTelegramPwd(telegramPwd string) Option {
	return func(o *telegramMessage) { o.telegramPwd = telegramPwd }
}

// 文本
func WithText(text string) Option {
	return func(o *telegramMessage) { o.text = text }
}

// 链接地址
func WithURL(url string) Option {
	return func(o *telegramMessage) { o.url = url }
}

// 文件类型(1=图片,2=视频)
func WithMsgType(msgType tgtypes.RobotMsgType) Option {
	return func(o *telegramMessage) { o.msgType = msgType }
}

// 是否开调式
func WithDebug(debug bool) Option {
	return func(o *telegramMessage) { o.debug = debug }
}

// 是否要发送到群里
func WithMessageThreadID(messageThreadID int64) Option {
	return func(o *telegramMessage) { o.messageThreadID = messageThreadID }
}

// ParseMode
func WithParseMode(parseMode tgtypes.ParseMode) Option {
	return func(o *telegramMessage) { o.parseMode = parseMode.String() }
}

// 文件名
func WithCustomFileName(customFileName string) Option {
	return func(o *telegramMessage) { o.customFileName = customFileName }
}
