package tgSet

import (
	"github.com/xbaseio/xbase/utils/xtelegram/telegram/telegram"
	"github.com/xbaseio/xbase/utils/xtelegram/telegram/types"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"

	"github.com/xbaseio/xbase/utils/xconv"
)

func SetChatMenuButton(botToken, username string, chatID int64) error {

	if botToken == "" {
		return nil
	}

	botApi, err := telegram.New(botToken)
	if err != nil {
		xlog.Logger().Error("log event", zap.Error(err))
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
		xlog.Logger().Error("log event", zap.Error(err))
		return err
	}
	xlog.Logger().Warn("botToken,botToken", zap.Any("botToken", botToken), zap.Any("rst", rst))

	return nil

}
