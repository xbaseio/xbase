package xtelegram

import (
	tgtypes "github.com/xbaseio/xbase/utils/xtelegram/tg-types"
)

type XTelegramMessage struct {
	botToken string //渠道code
	msg      *telegramMessage
}
type telegramMessage struct {
	telegramPwd string
	//keyboard        *tgbutton.TelegramButton //按钮
	url             string               //图片
	msgType         tgtypes.RobotMsgType //文件类型(1=图片,2=视频)
	text            string               //文本内容
	debug           bool
	messageThreadID int64  //是否要发送到群里
	parseMode       string //编码方式
	customFileName  string //文件名
}
