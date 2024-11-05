package main

import (
	"context"
	"os"

	"github.com/nekomeowww/tgo"
)

func main() {
	bot, err := tgo.NewBot(
		tgo.WithToken(os.Getenv("TELEGRAM_BOT_TOKEN")),
	)
	if err != nil {
		panic(err)
	}

	bot.OnCommand("ping", nil, tgo.NewHandler(func(ctx *tgo.Context) (tgo.Response, error) {
		return ctx.NewMessage("pong"), nil
	}))

	bot.Bootstrap(context.TODO())
}
