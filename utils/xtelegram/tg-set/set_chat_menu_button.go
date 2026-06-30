package tgSet

import (
	"github.com/xbaseio/xbase/utils/xtelegram/telegram/telegram"
	"github.com/xbaseio/xbase/utils/xtelegram/telegram/types"

	"github.com/xbaseio/xbase/log"
	"github.com/xbaseio/xbase/utils/xconv"
)

func SetChatMenuButton(botToken, username string, chatID int64) error {

	if botToken == "" {
		return nil
	}

	botApi, err := telegram.New(botToken)
	if err != nil {
		log.Errorf("%v", err)
		return nil
	}

	rst, err := botApi.SetChatMenuButton(&types.SetChatMenuButton{
		ChatID:     chatID,               // required. use for chat|channel as int
		ChatIDStr:  xconv.String(chatID), // required. use for chat|channel as string
		Username:   username,             // required. use for chat|channel
		MenuButton: botApi.NewMenuButtonWebApp("test", "https://www.baidu.com"),
	})

	//6867997452:AAFYZXHAC_TDvcfBiYto2ShutRSiUcboa04
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	log.Warnf("botToken,botToken:%v %#v", botToken, rst)

	return nil

}
