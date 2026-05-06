package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/xbaseio/xbase/utils/xtelegram"
	"github.com/xbaseio/xbase/utils/xtelegram/types"
)

// Use inline mode @botname some_text

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
	if update.InlineQuery == nil {
		return
	}

	results := []types.InlineQueryResult{
		&types.InlineQueryResultArticle{ID: "1", Title: "Foo 1", InputMessageContent: &types.InputTextMessageContent{MessageText: "foo 1"}},
		&types.InlineQueryResultArticle{ID: "2", Title: "Foo 2", InputMessageContent: &types.InputTextMessageContent{MessageText: "foo 2"}},
		&types.InlineQueryResultArticle{ID: "3", Title: "Foo 3", InputMessageContent: &types.InputTextMessageContent{MessageText: "foo 3"}},
	}

	b.AnswerInlineQuery(ctx, &xtelegram.AnswerInlineQueryParams{
		InlineQueryID: update.InlineQuery.ID,
		Results:       results,
	})
}
