package xtelegram

import (
	"fmt"

	"github.com/xbaseio/xbase/utils/expand"

	"github.com/xbaseio/xbase/mode"

	"github.com/xbaseio/xbase/utils/xtelegram/telegram/client"
	"github.com/xbaseio/xbase/utils/xtelegram/telegram/telegram"
	"github.com/xbaseio/xbase/utils/xtelegram/telegram/types"
	tgtypes "github.com/xbaseio/xbase/utils/xtelegram/tg-types"
)

func NewXTelegramMessage(botToken string, opts ...Option) (*XTelegramMessage, error) {
	if botToken == "" {
		return nil, fmt.Errorf("bot token is nil")
	}
	xMsg := &XTelegramMessage{
		botToken: botToken,
		msg:      &telegramMessage{},
	}
	for _, opt := range opts {
		opt(xMsg.msg)
	}
	return xMsg, nil
}

func (xMsg *XTelegramMessage) newApi() (*telegram.Api, error) {
	cfg := &client.Config{}
	if mode.IsTestMode() {
		cfg.Proxy = "http://127.0.0.1:7890"
		cfg.ForceV4 = true
	}

	botApi, err := telegram.NewWithCustomClient(xMsg.botToken, cfg)
	if err != nil {
		return nil, err
	}
	return botApi, nil
}

/*
	map[string]string{
		"customer": customer,
	})
*/
func (xMsg *XTelegramMessage) text(replaces map[string]string) string {

	if replaces == nil {
		return xMsg.msg.text
	}
	if len(replaces) == 0 {
		return xMsg.msg.text
	}
	return expand.Expand(xMsg.msg.text, replaces)
}
func (xMsg *XTelegramMessage) SendPhoto(chatID int64, replaces map[string]string) (*types.Message, error) {

	botApi, err := xMsg.newApi()
	if err != nil {
		return nil, err
	}
	botApi.Bot.Debug = xMsg.msg.debug

	if xMsg.msg.url != "" {
		message := botApi.NewSendPhoto()
		message.ChatID = chatID

		message.Photo = botApi.FileURL(xMsg.msg.url)

		message.Caption = xMsg.text(replaces)
		message.ReplyMarkup = xMsg.replyMarkup()
		if xMsg.msg.messageThreadID != 0 {
			message.MessageThreadID = xMsg.msg.messageThreadID
		}
		if xMsg.msg.parseMode != "" {
			message.ParseMode = xMsg.msg.parseMode
		}
		return botApi.SendPhoto(message)
	}
	return xMsg.SendTextMessage(chatID, replaces)

}
func (xMsg *XTelegramMessage) SendVideo(chatID int64, replaces map[string]string) (*types.Message, error) {

	botApi, err := xMsg.newApi()
	if err != nil {
		return nil, err
	}
	botApi.Bot.Debug = xMsg.msg.debug
	if xMsg.msg.url != "" {
		message := botApi.NewSendVideo()
		message.ChatID = chatID
		message.Video = botApi.FileURL(xMsg.msg.url)

		message.Caption = xMsg.text(replaces)
		message.ReplyMarkup = xMsg.replyMarkup()
		if xMsg.msg.messageThreadID != 0 {
			message.MessageThreadID = xMsg.msg.messageThreadID
		}
		if xMsg.msg.parseMode != "" {
			message.ParseMode = xMsg.msg.parseMode
		}
		return botApi.SendVideo(message)
	} else {
		return xMsg.SendTextMessage(chatID, replaces)
	}
}
func (xMsg *XTelegramMessage) SendDocument(chatID int64, replaces map[string]string) (*types.Message, error) {

	botApi, err := xMsg.newApi()
	if err != nil {
		return nil, err
	}
	botApi.Bot.Debug = xMsg.msg.debug
	if xMsg.msg.url != "" {
		message := botApi.NewSendDocument()
		message.ChatID = chatID
		message.Document = botApi.FileURL(xMsg.msg.url)
		message.Caption = xMsg.text(replaces)
		message.ReplyMarkup = xMsg.replyMarkup()
		message.CustomFileName = xMsg.msg.customFileName
		if xMsg.msg.messageThreadID != 0 {
			message.MessageThreadID = xMsg.msg.messageThreadID
		}
		if xMsg.msg.parseMode != "" {
			message.ParseMode = xMsg.msg.parseMode
		}
		return botApi.SendDocument(message)
	} else {
		return xMsg.SendTextMessage(chatID, replaces)
	}

}
func (xMsg *XTelegramMessage) SendTextMessage(chatID int64, replaces map[string]string) (*types.Message, error) {

	botApi, err := xMsg.newApi()
	if err != nil {
		return nil, err
	}
	botApi.Bot.Debug = xMsg.msg.debug

	message := botApi.NewSendMessage()
	message.ChatID = chatID
	message.Text = xMsg.text(replaces)
	message.ReplyMarkup = xMsg.replyMarkup()
	if xMsg.msg.messageThreadID != 0 {
		message.MessageThreadID = xMsg.msg.messageThreadID
	}
	if xMsg.msg.parseMode != "" {
		message.ParseMode = xMsg.msg.parseMode
	}
	return botApi.SendMessage(message)

}

func (xMsg *XTelegramMessage) replyMarkup() any {
	/*if xMsg.msg.keyboard == nil {
		return nil
	}
	return xMsg.msg.keyboard.Button(xMsg.msg.telegramPwd)
	*/
	return nil
}

func (xMsg *XTelegramMessage) SendMessage(chatID int64, replaces map[string]string) (*types.Message, error) {
	if xMsg == nil {
		return nil, nil
	}
	if xMsg.botToken == "" {
		return nil, nil
	}
	switch xMsg.msg.msgType {
	case tgtypes.RobotMsgTypePhoto:
		{
			return xMsg.SendPhoto(chatID, replaces)
		}
	case tgtypes.RobotMsgTypeVideo:
		{
			return xMsg.SendVideo(chatID, replaces)
		}
	}

	return xMsg.SendTextMessage(chatID, replaces)
}
func (xMsg *XTelegramMessage) EditMessageText(chatID int64, messageID int64, replaces map[string]string) (*types.Message, error) {
	botApi, err := xMsg.newApi()
	if err != nil {
		return nil, err
	}
	botApi.Bot.Debug = xMsg.msg.debug

	message := botApi.NewEditMessageText()
	message.ChatID = chatID
	message.MessageID = int(messageID)
	message.Text = xMsg.text(replaces)
	message.ReplyMarkup = xMsg.replyMarkup()

	if xMsg.msg.parseMode != "" {
		message.ParseMode = xMsg.msg.parseMode
	}
	return botApi.EditMessageText(message)
}
func (xMsg *XTelegramMessage) DeleteMessage(chatID int64, messageID int) (bool, error) {
	botApi, err := xMsg.newApi()
	if err != nil {
		return false, err
	}
	botApi.Bot.Debug = xMsg.msg.debug
	return botApi.DeleteMessage(&types.DeleteMessage{
		ChatID:    chatID,
		MessageID: messageID,
	})
}

// 设置额外信息
func (xMsg *XTelegramMessage) SetExtraCallBackData(orderID string) error {
	/*
		if xMsg == nil {
			return nil
		}
		if xMsg.msg == nil {
			return nil
		}
		if xMsg.msg.keyboard == nil {
			return nil
		}
		if xMsg.msg.keyboard.CmdKind != tgtypes.CmdKind_InlineKeyboard {
			return nil
		}
		if xMsg.msg.keyboard.InlineKeyboard == nil {
			return nil
		}
		if orderID == "" {
			return nil
		}
		for lineID, lines := range xMsg.msg.keyboard.InlineKeyboard.InlineKeyboard {
			for rowId, row := range lines {
				button := tgtypes.StringToXTelegramButton(row.CallbackData)
				if button.IsAddOrder() {
					xMsg.msg.keyboard.InlineKeyboard.InlineKeyboard[lineID][rowId].CallbackData += orderID
					if len(xMsg.msg.keyboard.InlineKeyboard.InlineKeyboard[lineID][rowId].CallbackData) > 64 {
						return errors.NewError(code.ButtonInvalid, xMsg.msg.keyboard.InlineKeyboard.InlineKeyboard[lineID][rowId].CallbackData)
					}
				}

			}
		}
	*/
	return nil
}
