package tgSet

import (
	"github.com/xbaseio/xbase/utils/xtelegram/telegram/telegram"
	"github.com/xbaseio/xbase/utils/xtelegram/telegram/types"
	"github.com/xbaseio/xbase/xlog"
)

func SetStickerEmojiList(botToken string) error {

	if botToken == "" {
		return nil
	}

	botApi, err := telegram.New(botToken)
	if err != nil {
		xlog.Sugar().Errorf("%v", err)
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
		xlog.Sugar().Errorf("%v", err)
		return err
	}
	xlog.Sugar().Warnf("botToken,botToken:%v %#v", botToken, rst)

	return nil

}
