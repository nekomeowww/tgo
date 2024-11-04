package queue

import (
	"context"

	"github.com/redis/rueidis"
)

var _ Queue = (*RueidisQueue)(nil)

type RueidisQueue struct {
	rueidis rueidis.Client
}

func NewRueidisQueue(client rueidis.Client) *RueidisQueue {
	return &RueidisQueue{
		rueidis: client,
	}
}

func (q *RueidisQueue) Push(ctx context.Context, group string, t string) error {
	lpushCmd := q.rueidis.B().
		Lpush().
		Key(group).
		Element(t).
		Build()

	exCmd := q.rueidis.B().
		Expire().
		Key(group).
		Seconds(24 * 60 * 60).
		Build()

	res := q.rueidis.DoMulti(ctx, lpushCmd, exCmd)
	for _, v := range res {
		if v.Error() != nil {
			return v.Error()
		}
	}

	return nil
}

func (q *RueidisQueue) Pop(ctx context.Context, group string) (string, error) {
	lrangeCmd := q.rueidis.B().
		Rpop().
		Key(group).
		Count(1).
		Build()

	elems, err := q.rueidis.Do(ctx, lrangeCmd).ToString()
	if err != nil {
		if rueidis.IsRedisNil(err) {
			return "", nil
		}

		return "", err
	}

	return elems, nil
}

func (q *RueidisQueue) PopAll(ctx context.Context, group string) ([]string, error) {
	lrangeCmd := q.rueidis.B().
		Lrange().
		Key(group).
		Start(0).
		Stop(-1).
		Build()

	elems, err := q.rueidis.Do(context.Background(), lrangeCmd).AsStrSlice()
	if err != nil {
		return make([]string, 0), nil
	}
	if len(elems) == 0 {
		return make([]string, 0), nil
	}

	delCmd := q.rueidis.B().
		Del().
		Key(group).
		Build()

	res := q.rueidis.Do(context.Background(), delCmd)
	if res.Error() != nil {
		return nil, res.Error()
	}

	return elems, nil
}
