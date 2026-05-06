package main

import (
	"context"
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
	if update.Message == nil {
		return
	}

	m, errSend := b.SendMessage(ctx, &xtelegram.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
	if errSend != nil {
		fmt.Printf("error sending message: %v\n", errSend)
		return
	}

	time.Sleep(time.Second * 2)

	_, errEdit := b.EditMessageText(ctx, &xtelegram.EditMessageTextParams{
		ChatID:    m.Chat.ID,
		MessageID: m.ID,
		Text:      "New Message!",
	})
	if errEdit != nil {
		fmt.Printf("error edit message: %v\n", errEdit)
		return
	}
}
