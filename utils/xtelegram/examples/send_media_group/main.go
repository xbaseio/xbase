package main

import (
	"bytes"
	"context"
	"embed"
	"os"
	"os/signal"

	"github.com/xbaseio/xbase/utils/xtelegram"
	"github.com/xbaseio/xbase/utils/xtelegram/types"
)

// Send any text message to the bot after the bot has been started

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []xtelegram.Option{
		xtelegram.WithDefaultHandler(handler),
	}

	b, err := xtelegram.New(os.Getenv("EXAMPLE_TELEGRAM_BOT_TOKEN"), opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	b.Start(ctx)
}

//go:embed images
var images embed.FS

func handler(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
	fileDataFacebook, _ := images.ReadFile("images/facebook.png")
	fileDataYoutube, _ := images.ReadFile("images/youtube.png")

	media1 := &types.InputMediaPhoto{
		Media:   "https://telegram.org/img/t_logo.png",
		Caption: "Telegram Logo",
	}

	media2 := &types.InputMediaPhoto{
		Media:           "attach://facebook.png",
		Caption:         "Facebook Logo",
		MediaAttachment: bytes.NewReader(fileDataFacebook),
	}

	media3 := &types.InputMediaPhoto{
		Media:           "attach://youtube.png",
		Caption:         "Youtube Logo",
		MediaAttachment: bytes.NewReader(fileDataYoutube),
	}

	params := &xtelegram.SendMediaGroupParams{
		ChatID: update.Message.Chat.ID,
		Media: []types.InputMedia{
			media1,
			media2,
			media3,
		},
	}

	b.SendMediaGroup(ctx, params)
}
