package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/xbaseio/xbase/utils/xtelegram"
	"github.com/xbaseio/xbase/utils/xtelegram/types"
)

// Send any text message to the bot after the bot has been started

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []xtelegram.Option{
		xtelegram.WithDefaultHandler(handler),
		xtelegram.WithSkipGetMe(),
	}

	b, err := xtelegram.New(os.Getenv("EXAMPLE_TELEGRAM_BOT_TOKEN"), opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	user, err := b.GetMe(ctx)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}
	botUsername = user.Username

	b.Start(ctx)
}

//go:embed images
var images embed.FS
var botUsername string

func handler(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
	fileContent, _ := images.ReadFile("images/telegram.png")

	inputSticker1 := types.InputSticker{
		Sticker:   "https://github.com/xbaseio/xbase/utils/xtelegram/blob/main/examples/create_new_sticker_set/images/telegram.png?raw=true",
		Format:    "static",
		EmojiList: []string{"1️⃣"},
	}

	inputSticker2 := types.InputSticker{
		Sticker:           "attach://telegram.png",
		Format:            "static",
		EmojiList:         []string{"2️⃣"},
		StickerAttachment: bytes.NewReader(fileContent),
	}

	stickerSetName := fmt.Sprintf("Example%d_by_%s", time.Now().Unix(), botUsername)
	params := &xtelegram.CreateNewStickerSetParams{
		UserID: update.Message.Chat.ID,
		Name:   stickerSetName,
		Title:  "Example sticker set",
		Stickers: []types.InputSticker{
			inputSticker1,
			inputSticker2,
		},
	}

	_, err := b.CreateNewStickerSet(ctx, params)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	stickerSet, err := b.GetStickerSet(ctx, &xtelegram.GetStickerSetParams{Name: stickerSetName})
	if err != nil {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}
	_, err = b.SendSticker(
		ctx,
		&xtelegram.SendStickerParams{
			ChatID: update.Message.Chat.ID,
			Sticker: &types.InputFileString{
				Data: stickerSet.Stickers[0].FileID,
			},
		},
	)
	if err != nil {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}
}
