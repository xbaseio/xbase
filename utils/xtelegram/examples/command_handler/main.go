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

	b.RegisterHandler(xtelegram.HandlerTypeMessageText, "foo", xtelegram.MatchTypeCommand, fooHandler)
	b.RegisterHandler(xtelegram.HandlerTypeMessageText, "bar", xtelegram.MatchTypeCommandStartOnly, barHandler)

	b.Start(ctx)
}

func fooHandler(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
	b.SendMessage(ctx, &xtelegram.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Caught *foo*",
		ParseMode: types.ParseModeMarkdown,
	})
}

func barHandler(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
	b.SendMessage(ctx, &xtelegram.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Caught *bar*",
		ParseMode: types.ParseModeMarkdown,
	})
}

func defaultHandler(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
	b.SendMessage(ctx, &xtelegram.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Say message with `/foo` anywhere or with `/bar` at start of the message",
		ParseMode: types.ParseModeMarkdown,
	})
}
