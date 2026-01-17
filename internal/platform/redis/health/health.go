package healthredis

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisPinger struct {
	rdb *redis.Client
}

func NewRedisPinger(rdb *redis.Client) RedisPinger {
	return RedisPinger{rdb: rdb}
}

func (p RedisPinger) Ping(ctx context.Context) error {
	return p.rdb.Ping(ctx).Err()
}
