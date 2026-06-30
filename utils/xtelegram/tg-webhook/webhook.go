package tgwebhook

import (
	"github.com/xbaseio/xbase/log"
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
	//6867997452:AAFYZXHAC_TDvcfBiYto2ShutRSiUcboa04
	log.Warnf("channelCode:%v CallBack:%v, %#v:%v", channelCode, webHookUrl, res, err)
	if err != nil {
		return err
	}
	//SetChatMenuButton(botToken)
	//SetMyCommands(botToken)
	//SetStickerEmojiList(botToken)
	return nil

}
