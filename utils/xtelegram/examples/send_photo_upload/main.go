package main

import (
	"bytes"
	"context"
	"fmt"
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

func handler(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
	fileData, errReadFile := os.ReadFile("./examples/send_photo_upload/facebook.png")
	if errReadFile != nil {
		fmt.Printf("error read file, %v\n", errReadFile)
		return
	}

	params := &xtelegram.SendPhotoParams{
		ChatID:  update.Message.Chat.ID,
		Photo:   &types.InputFileUpload{Filename: "facebook.png", Data: bytes.NewReader(fileData)},
		Caption: "New uploaded Facebook logo",
	}

	b.SendPhoto(ctx, params)
}
