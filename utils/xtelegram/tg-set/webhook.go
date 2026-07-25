package tgSet

import (
	"go.uber.org/zap"
)

func SetWebHook(botToken, channelCode, webHookUrl string) error {

	if botToken == "" {
		return nil
	}

	client := NewClient(botToken)

	data := make(map[string]string)

	data["url"] = webHookUrl
	data["secret_token"] = channelCode
	var res any
	err := client.Get("/setWebhook", data, res)
	zap.
		//6867997452:AAFYZXHAC_TDvcfBiYto2ShutRSiUcboa04
		S().Warnf("channelCode:%v CallBack:%v, %#v:%v", channelCode, webHookUrl, res, err)
	if err != nil {
		return err
	}
	return nil

}
