package tgSet

import (
	"github.com/xbaseio/xbase/utils/xtelegram/telegram/telegram"
	"github.com/xbaseio/xbase/utils/xtelegram/telegram/types"
	"github.com/xbaseio/xbase/xlog"
	"go.uber.org/zap"
)

func SetStickerEmojiList(botToken string) error {

	if botToken == "" {
		return nil
	}

	botApi, err := telegram.New(botToken)
	if err != nil {
		xlog.Logger().Error("log event", zap.Error(err))
		return nil
	}

	/*rst, err := botApi.SetStickerKeywords(&types.SetStickerKeywords{
		Sticker:  "Abc",
		Keywords: []string{"💹TRX闪兑", "💹TRX闪兑01"},
	})
	*/
	rst, err := botApi.SetStickerEmojiList(&types.SetStickerEmojiList{
		Sticker:   "rand55566",
		EmojiList: []string{"🏧bbc", "💹"},
	})

	if err != nil {
		xlog.Logger().Error("log event", zap.Error(err))
		return err
	}
	xlog.Logger().Warn("botToken,botToken", zap.Any("botToken", botToken), zap.Any("rst", rst))

	return nil

}
