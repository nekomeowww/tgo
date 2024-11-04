package tgo

import (
	"encoding/json"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/nekomeowww/tgo/pkg/storage/queue"
	"github.com/nekomeowww/tgo/pkg/storage/ttlcache"
	"github.com/nekomeowww/xo/logger"
	"github.com/redis/rueidis"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestBindFromCallbackQueryData(t *testing.T) {
	t.Run("InMemory", func(t *testing.T) {
		logger, err := logger.NewLogger(logger.WithLevel(zapcore.DebugLevel), logger.WithAppName("tgo"), logger.WithNamespace("nekomeowww"))
		require.NoError(t, err)

		data := struct {
			Hello string `json:"hello"`
		}{
			Hello: "world",
		}

		ctx := NewContext(nil, &BotAPI{
			logger:   logger,
			queue:    queue.NewInMemoryQueue(),
			ttlcache: ttlcache.NewInMemoryTTLCache(),
		}, tgbotapi.Update{}, logger, nil)
		ctx.withCallbackQueryActionData(string(lo.Must(json.Marshal(data))))

		var dst struct {
			Hello string `json:"hello"`
		}

		err = ctx.BindFromCallbackQueryData(&dst)
		require.NoError(t, err)
		assert.Equal(t, data, dst)
	})

	t.Run("Rueidis", func(t *testing.T) {
		logger, err := logger.NewLogger(logger.WithLevel(zapcore.DebugLevel), logger.WithAppName("tgo"), logger.WithNamespace("nekomeowww"))
		require.NoError(t, err)

		c, err := rueidis.NewClient(rueidis.ClientOption{
			InitAddress:  []string{"localhost:6379"},
			DisableCache: true,
		})
		require.NoError(t, err)

		data := struct {
			Hello string `json:"hello"`
		}{
			Hello: "world",
		}

		ctx := NewContext(nil, &BotAPI{
			logger:   logger,
			queue:    queue.NewRueidisQueue(c),
			ttlcache: ttlcache.NewRueidisTTLCache(c),
		}, tgbotapi.Update{}, logger, nil)
		ctx.withCallbackQueryActionData(string(lo.Must(json.Marshal(data))))

		var dst struct {
			Hello string `json:"hello"`
		}

		err = ctx.BindFromCallbackQueryData(&dst)
		require.NoError(t, err)
		assert.Equal(t, data, dst)
	})
}
