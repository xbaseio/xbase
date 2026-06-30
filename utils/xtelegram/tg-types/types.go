package tgtypes

import tgconfig "github.com/xbaseio/xbase/utils/xtelegram/telegram/config"

type ParseMode string

const (
	ModeNone       ParseMode = ""
	ModeMarkdown   ParseMode = tgconfig.ModeMarkdown
	ModeMarkdownV2 ParseMode = tgconfig.ModeMarkdownV2
	ModeHTML       ParseMode = tgconfig.ModeHTML
)

func (pm ParseMode) String() string {
	return string(pm)
}

type ButtonKind string

const (
	ButtonType_InlineKeyboardButtonURL               ButtonKind = "InlineKeyboardButtonURL"      //跳转网页
	ButtonKind_InlineKeyboardCallbackData            ButtonKind = "InlineKeyboardCallbackData"   //跳转应用
	ButtonKind_InlineKeyboardWebApp                  ButtonKind = "InlineKeyboardWebApp"         //按钮
	ButtonKind_InlineKeyboardButtonLoginURL          ButtonKind = "InlineKeyboardButtonLoginURL" //按钮
	ButtonKind_InlineKeyboardButtonSwitch            ButtonKind = "InlineKeyboardButtonSwitch"
	ButtonKind_InlineKeyboardButtonSwitchCurrentChat ButtonKind = "InlineKeyboardButtonSwitchCurrentChat"
	ButtonKind_InlineKeyboardButtonSwitchChosenChat  ButtonKind = "InlineKeyboardButtonSwitchChosenChat"
)

type CmdKind string

const (
	CmdKind_None           CmdKind = "none"           //无按钮
	CmdKind_InlineKeyboard CmdKind = "InlineKeyboard" //内联按钮
	CmdKind_KeyboardMarkup CmdKind = "KeyboardMarkup" //跳转应用

)

type RobotMsg int8

const (
	RobotMsgLoop      RobotMsg = 1 //循环推送
	RobotMsgTime      RobotMsg = 2 //定时推送
	RobotMsgForthwith RobotMsg = 3 //即时推送
)

type RobotMsgStatus int8

const (
	RobotMsgStatusNotSend RobotMsgStatus = 1 //未发送
	RobotMsgStatusSend    RobotMsgStatus = 2 //已发送
)

type RobotMsgType int

const (
	RobotMsgTypePhoto    RobotMsgType = 1 //图片
	RobotMsgTypeVideo    RobotMsgType = 2 //视频
	RobotMsgTypeText     RobotMsgType = 3 //文本
	RobotMsgTypeDocument RobotMsgType = 4 //文档
)
