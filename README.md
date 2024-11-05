# `tgo`

[![Go Reference](https://pkg.go.dev/badge/github.com/nekomeowww/tgo.svg)](https://pkg.go.dev/github.com/nekomeowww/tgo)
![](https://github.com/nekomeowww/tgo/actions/workflows/ci.yml/badge.svg)
[![](https://goreportcard.com/badge/github.com/nekomeowww/tgo)](https://goreportcard.com/report/github.com/nekomeowww/tgo)

Telegram Bot API for Go, with [@nekomeowww](https://github.com/nekomeowww)'s flavor, born from varies of Telegram Bot projects:

- [`insights-bot`](https://github.com/nekomeowww/insights-bot)
- [`perobot`](https://github.com/nekomeowww/perobot)
- [`factorio-chat-bridge`](https://github.com/nekomeowww/factorio-chat-bridge)

## Features

- [x] 🎺 Wrapper for commands, callback queries, inline queries
- [x] 🎆 Any-length callback query data, no more 64-bytes fighting
- [x] 🎯 Battle-tested dispatcher for each supported updates
- [x] 👮 Middleware support (guard, permission check, etc.)
- [x] 🌍 Opt-in i18n support
- [x] 🚀 Easy to use, easy to extend
- [x] 🍱 Useful helpers for permission check, message handling, error handling
- [x] 📦 Dependency injection friendly
- [x] 📚 More examples and documentation
- [x] 🛜 Out of the box support for webhooks & polling

## 🤠 Spec & Documentation

GoDoc: [https://godoc.org/github.com/nekomeowww/tgo](https://godoc.org/github.com/nekomeowww/tgo)

## Usage

```shell
go get -u github.com/nekomeowww/tgo
```

```go
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
```

if you use `uber/fx` too, you can follow this example:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nekomeowww/fo"
	"github.com/nekomeowww/tgo"

	"go.uber.org/fx"
)

func NewBot() func() (*tgo.Bot, error) {
	return func() (*tgo.Bot, error) {
		bot, err := tgo.NewBot(tgo.WithToken(os.Getenv("TELEGRAM_BOT_TOKEN")))
		if err != nil {
			return nil, err
		}

		bot.OnCommand("ping", nil, tgo.NewHandler(func(ctx *tgo.Context) (tgo.Response, error) {
    		return ctx.NewMessage("pong"), nil
    	}))

		return bot, nil
	}
}

func Run() func(fx.Lifecycle, *tgo.Bot) {
	return func(lifecycle fx.Lifecycle, bot *tgo.Bot) {
		lifecycle.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					_ = bot.Start(ctx)
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return bot.Stop(ctx)
			},
		})
	}
}

func main() {
	app := fx.New(
		fx.Provide(NewBot()),
		fx.Invoke(Run()),
	)

	app.Run()

	stopCtx, stopCtxCancel := context.WithTimeout(context.Background(), time.Second*15)
	defer stopCtxCancel()

	if err := app.Stop(stopCtx); err != nil {
		log.Fatal(err)
	}
}
```
