package ttlcache

import (
	"context"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/samber/mo"
)

type InMemoryTTLCache struct {
	mutex sync.Mutex

	mCache      map[time.Duration]*cache.Cache
	mKeyMapping map[string]*cache.Cache
}

func NewInMemoryTTLCache() *InMemoryTTLCache {
	return &InMemoryTTLCache{
		mCache:      make(map[time.Duration]*cache.Cache),
		mKeyMapping: make(map[string]*cache.Cache),
	}
}

func (c *InMemoryTTLCache) Get(_ context.Context, key string) (mo.Option[string], error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if cache, ok := c.mKeyMapping[key]; ok {
		if value, found := cache.Get(key); found {
			return mo.Some(value.(string)), nil
		}
	}

	return mo.None[string](), nil
}

func (c *InMemoryTTLCache) Set(_ context.Context, key string, value string, ttl time.Duration) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if foundCache, ok := c.mKeyMapping[key]; ok {
		foundCache.Set(key, value, ttl)
	} else {
		cache := cache.New(ttl, time.Second)
		cache.Set(key, value, ttl)
		c.mCache[ttl] = cache
		c.mKeyMapping[key] = cache
	}

	return nil
}
