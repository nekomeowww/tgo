package ttlcache

import (
	"context"
	"time"

	"github.com/redis/rueidis"
	"github.com/samber/mo"
)

type RueidisTTLCache struct {
	rueidis rueidis.Client
}

func NewRueidisTTLCache(client rueidis.Client) *RueidisTTLCache {
	return &RueidisTTLCache{
		rueidis: client,
	}
}

func (c *RueidisTTLCache) Get(ctx context.Context, key string) (mo.Option[string], error) {
	getCmd := c.rueidis.B().
		Get().
		Key(key).
		Build()

	str, err := c.rueidis.Do(context.Background(), getCmd).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return mo.None[string](), nil
		}

		return mo.None[string](), err
	}

	return mo.Some(str), nil
}

func (c *RueidisTTLCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	setCmd := c.rueidis.B().
		Set().
		Key(key).
		Value(value).
		ExSeconds(int64(ttl.Seconds())).
		Build()

	err := c.rueidis.Do(context.Background(), setCmd).Error()
	if err != nil {
		return err
	}

	return nil
}
