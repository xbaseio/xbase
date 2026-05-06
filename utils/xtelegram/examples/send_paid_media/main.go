package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"os"
	"os/signal"

	"github.com/xbaseio/xbase/utils/xtelegram"
	"github.com/xbaseio/xbase/utils/xtelegram/types"
)

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

	if update.ChannelPost == nil {
		fmt.Printf("expect channel post\n")
		return
	}

	chatID := update.ChannelPost.Chat.ID

	media1 := &types.InputPaidMediaPhoto{
		Media: "https://telegram.org/img/t_logo.png",
	}

	media2 := &types.InputPaidMediaPhoto{
		Media:           "attach://facebook.png",
		MediaAttachment: bytes.NewReader(fileDataFacebook),
	}

	media3 := &types.InputPaidMediaPhoto{
		Media:           "attach://youtube.png",
		MediaAttachment: bytes.NewReader(fileDataYoutube),
	}

	params := &xtelegram.SendPaidMediaParams{
		ChatID:    chatID,
		StarCount: 10,
		Media: []types.InputPaidMedia{
			media1,
			media2,
			media3,
		},
	}

	_, err := b.SendPaidMedia(ctx, params)
	if err != nil {
		fmt.Printf("%+v\n", err)
	}
}
