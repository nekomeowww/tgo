package queue

import (
	"context"
	"sync"
)

var _ Queue = (*InMemoryQueue)(nil)

type InMemoryQueue struct {
	mutex sync.Mutex

	items map[string][]string
}

func NewInMemoryQueue() *InMemoryQueue {
	return &InMemoryQueue{
		items: make(map[string][]string, 0),
	}
}

func (q *InMemoryQueue) Push(_ context.Context, group string, data string) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.items[group] = append(q.items[group], data)

	return nil
}

func (q *InMemoryQueue) Pop(_ context.Context, group string) (string, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if len(q.items[group]) == 0 {
		return "", nil
	}

	data := q.items[group][0]
	q.items[group] = q.items[group][1:]

	return data, nil
}

func (q *InMemoryQueue) PopAll(_ context.Context, group string) ([]string, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	data := append([]string{}, q.items[group]...)
	q.items[group] = make([]string, 0)

	return data, nil
}
