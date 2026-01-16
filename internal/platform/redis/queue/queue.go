package queue

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisQueue struct {
	rdb    *redis.Client
	stream string
}

func New(rdb *redis.Client, stream string) *RedisQueue {
	return &RedisQueue{rdb: rdb, stream: stream}
}

type Item struct {
	EventType string
	Payload   string
	OutboxID  int64
}

func (q *RedisQueue) EnqueueBatch(ctx context.Context, items []Item) error {
	if len(items) == 0 {
		return nil
	}

	pipe := q.rdb.Pipeline()
	for _, it := range items {
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: q.stream,
			Values: map[string]any{
				"type":      it.EventType,
				"body":      it.Payload,
				"outbox_id": it.OutboxID,
			},
		})
	}
	_, err := pipe.Exec(ctx)
	return err
}
