package queue

import "context"

type Queue interface {
	Push(context.Context, string, string) error
	Pop(context.Context, string) (string, error)
	PopAll(context.Context, string) ([]string, error)
}
