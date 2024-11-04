package tgo

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/nekomeowww/tgo/pkg/redis"
)

func (b *BotAPI) couldCountRateLimitFor(key string, rate int64, perDuration time.Duration) (int64, bool, error) {
	if perDuration <= 0 {
		return 0, true, nil
	}

	countedRateStr, err := b.ttlcache.Get(context.Background(), key)
	if err != nil {
		return 0, false, err
	}

	countedRate, _ := strconv.ParseInt(countedRateStr.OrEmpty(), 10, 64)
	if countedRate >= rate {
		return countedRate, false, nil
	}

	countedRate++

	err = b.ttlcache.Set(context.Background(), key, fmt.Sprintf("%d", countedRate), time.Duration(int64(perDuration/time.Second))*time.Second)
	if err != nil {
		return countedRate, false, err
	}

	return countedRate, true, nil
}

func (b *BotAPI) RateLimitForCommand(chatID int64, command string, rate int64, perDuration time.Duration) (int64, bool, error) {
	// TODO: telegram is static and hardcoded, should be dynamic and constant with specific type when integrated more platforms
	return b.couldCountRateLimitFor(redis.CommandRateLimitLock2.Format(command, "telegram", strconv.FormatInt(chatID, 10)), rate, perDuration)
}
