package tgwebhook

import (
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

func SetWebHook(botToken, channelCode, webHookUrl string) error {

	if botToken == "" {
		return nil
	}

	client := NewClient(botToken)

	data := make(map[string]string)

	data["url"] = webHookUrl
	if channelCode != "" {
		data["secret_token"] = channelCode
	}

	var res any
	err := client.Get("/setWebhook", data, res)
	xlog.Logger().Warn("channelCode: CallBack", zap.Any("channelCode", channelCode), zap.Any("webHookUrl", webHookUrl), zap.Any("res", res), zap.Error(err))
	if err != nil {
		return err
	}
	//SetChatMenuButton(botToken)
	//SetMyCommands(botToken)
	//SetStickerEmojiList(botToken)
	return nil

}
