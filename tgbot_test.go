package tgo

import (
	"encoding/json"
	"testing"

	"github.com/nekomeowww/tgo/pkg/storage/queue"
	"github.com/nekomeowww/tgo/pkg/storage/ttlcache"
	"github.com/nekomeowww/xo/logger"
	"github.com/redis/rueidis"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestAssignOneCallbackQueryData(t *testing.T) {
	t.Run("InMemory", func(t *testing.T) {
		data := struct {
			Hello string `json:"hello"`
		}{
			Hello: "world",
		}

		logger, err := logger.NewLogger(logger.WithLevel(zapcore.DebugLevel), logger.WithAppName("tgo"), logger.WithNamespace("nekomeowww"))
		require.NoError(t, err)

		bot := BotAPI{logger: logger, queue: queue.NewInMemoryQueue(), ttlcache: ttlcache.NewInMemoryTTLCache()}

		callbackQueryData, err := bot.AssignOneCallbackQueryData("test", data)
		require.NoError(t, err)

		routeHash, dataHash := bot.routeHashAndActionHashFromData(callbackQueryData)
		require.NotEmpty(t, routeHash)
		require.NotEmpty(t, dataHash)

		dataStr, err := bot.fetchCallbackQueryActionData("test", dataHash)
		require.NoError(t, err)

		assert.Equal(t, string(lo.Must(json.Marshal(data))), dataStr)
	})

	t.Run("Rueidis", func(t *testing.T) {
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

		logger, err := logger.NewLogger(logger.WithLevel(zapcore.DebugLevel), logger.WithAppName("tgo"), logger.WithNamespace("nekomeowww"))
		require.NoError(t, err)

		bot := BotAPI{logger: logger, queue: queue.NewRueidisQueue(c), ttlcache: ttlcache.NewRueidisTTLCache(c)}

		callbackQueryData, err := bot.AssignOneCallbackQueryData("test", data)
		require.NoError(t, err)

		routeHash, dataHash := bot.routeHashAndActionHashFromData(callbackQueryData)
		require.NotEmpty(t, routeHash)
		require.NotEmpty(t, dataHash)

		dataStr, err := bot.fetchCallbackQueryActionData("test", dataHash)
		require.NoError(t, err)

		assert.Equal(t, string(lo.Must(json.Marshal(data))), dataStr)
	})
}
