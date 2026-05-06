package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/xbaseio/xbase/utils/xtelegram"
	"github.com/xbaseio/xbase/utils/xtelegram/types"
)

// Send any text message to the bot after the bot has been started

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []xtelegram.Option{
		xtelegram.WithMiddlewares(showMessageWithUserID, showMessageWithUserName),
		xtelegram.WithDefaultHandler(handler),
	}

	b, err := xtelegram.New(os.Getenv("EXAMPLE_TELEGRAM_BOT_TOKEN"), opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	b.RegisterHandler(xtelegram.HandlerTypeCallbackQueryData, "", xtelegram.MatchTypeExact, func(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
		log.Printf("callback query data: %s", update.CallbackQuery.Data)
	}, singleFlight)

	b.Start(ctx)
}

func showMessageWithUserID(next xtelegram.HandlerFunc) xtelegram.HandlerFunc {
	return func(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
		if update.Message != nil {
			log.Printf("%d say: %s", update.Message.From.ID, update.Message.Text)
		}
		next(ctx, b, update)
	}
}

func showMessageWithUserName(next xtelegram.HandlerFunc) xtelegram.HandlerFunc {
	return func(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
		if update.Message != nil {
			log.Printf("%s say: %s", update.Message.From.FirstName, update.Message.Text)
		}
		next(ctx, b, update)
	}
}

// singleFlight is a middleware that ensures that only one callback query is processed at a time.
func singleFlight(next xtelegram.HandlerFunc) xtelegram.HandlerFunc {
	sf := sync.Map{}
	return func(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
		if update.CallbackQuery != nil {
			key := update.CallbackQuery.Message.Message.ID
			if _, loaded := sf.LoadOrStore(key, struct{}{}); loaded {
				return
			}
			defer sf.Delete(key)
			next(ctx, b, update)
		}
	}
}

func handler(ctx context.Context, b *xtelegram.Bot, update *types.Update) {
	b.SendMessage(ctx, &xtelegram.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}
