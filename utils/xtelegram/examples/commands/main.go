package main

import (
	"context"
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
		xtelegram.WithDefaultHandler(defaultHandler),
	}

	b, err := xtelegram.New(os.Getenv("EXAMPLE_TELEGRAM_BOT_TOKEN"), opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	b.RegisterHandler(xtelegram.HandlerTypeMessageText, "/hello", xtelegram.MatchTypeExact, helloHandler)

	b.Start(ctx)
}

func helloHandler(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
	b.SendMessage(ctx, &xtelegram.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Hello, *" + xtelegram.EscapeMarkdown(update.Message.From.FirstName) + "*",
		ParseMode: types.ParseModeMarkdown,
	})
}

func defaultHandler(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
	b.SendMessage(ctx, &xtelegram.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Say /hello",
	})
}
