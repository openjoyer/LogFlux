package ratelimit

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	limitKeyPrefix = "ratelimit:project"
	ttlSeconds     = 5
)

type Limiter struct {
	redisClient *redis.Client
	limit       int
}

func NewLimiter(client *redis.Client, limit int) *Limiter {
	return &Limiter{client, limit}
}

func (l *Limiter) Allow(ctx context.Context, projectID string) (bool, error) {
	second := strconv.FormatInt(time.Now().Unix(), 10)
	key := fmt.Sprintf("%s:%s:%s", limitKeyPrefix, projectID, second)

	value, err := l.increment(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	return int(value) <= l.limit, nil
}

func (l *Limiter) increment(ctx context.Context, key string) (int64, error) {
	var counter *redis.IntCmd

	_, err := l.redisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		counter = pipe.Incr(ctx, key)
		pipe.Expire(ctx, key, time.Duration(ttlSeconds)*time.Second)
		return nil
	})
	if err != nil {
		return 0, err
	}

	return counter.Val(), nil
}
