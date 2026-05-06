package main

import (
	"context"
	"fmt"
	"os"

	"github.com/xbaseio/xbase/utils/xtelegram"
)

func main() {
	b, err := xtelegram.New(os.Getenv("EXAMPLE_TELEGRAM_BOT_TOKEN"))
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	user, _ := b.GetMe(context.Background())

	fmt.Printf("User: %#v\n", user)
}
