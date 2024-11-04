package ttlcache

import (
	"context"
	"time"

	"github.com/samber/mo"
)

type TTLCache interface {
	Get(context.Context, string) (mo.Option[string], error)
	Set(context.Context, string, string, time.Duration) error
}
