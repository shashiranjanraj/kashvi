package queue

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	redisQueueKey   = "kashvi:queue:jobs"
	redisDelayedKey = "kashvi:queue:delayed"
)

// RedisDriver is a production-grade queue driver backed by Redis.
// Immediate jobs use LPUSH/BRPOP on a list.
// Delayed jobs use a sorted set scored by Unix timestamp.
type RedisDriver struct {
	rdb *redis.Client
	ctx context.Context
}

// NewRedisDriver creates a new Redis-backed queue driver.
// Pass the same *redis.Client used by pkg/cache.
func NewRedisDriver(rdb *redis.Client) *RedisDriver {
	d := &RedisDriver{rdb: rdb, ctx: context.Background()}
	go d.promoteDelayedJobs() // background ticker
	return d
}

// Push adds a job payload to the immediate queue (LPUSH).
func (d *RedisDriver) Push(payload []byte) error {
	if err := d.rdb.LPush(d.ctx, redisQueueKey, payload).Err(); err != nil {
		return fmt.Errorf("queue/redis: push: %w", err)
	}
	return nil
}

// Pop blocks until a job is available (BRPOP with 5s timeout).
func (d *RedisDriver) Pop(ctx context.Context) ([]byte, error) {
	result, err := d.rdb.BRPop(ctx, 5*time.Second, redisQueueKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // timeout — no jobs ready, normal
		}
		return nil, fmt.Errorf("queue/redis: pop: %w", err)
	}
	if len(result) < 2 {
		return nil, nil
	}
	return []byte(result[1]), nil
}

// PushDelayed schedules a job to run after delay using a Redis sorted set.
// The score is the Unix timestamp when the job should be promoted.
func (d *RedisDriver) PushDelayed(payload []byte, delay time.Duration) error {
	runAt := float64(time.Now().Add(delay).Unix())
	if err := d.rdb.ZAdd(d.ctx, redisDelayedKey, redis.Z{
		Score:  runAt,
		Member: string(payload),
	}).Err(); err != nil {
		return fmt.Errorf("queue/redis: push delayed: %w", err)
	}
	return nil
}

// promoteDelayedJobs moves jobs whose scheduled time has passed into the main queue.
// Runs every second in the background.
func (d *RedisDriver) promoteDelayedJobs() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := strconv.FormatInt(time.Now().Unix(), 10)
		jobs, err := d.rdb.ZRangeByScore(d.ctx, redisDelayedKey, &redis.ZRangeBy{
			Min: "-inf",
			Max: now,
		}).Result()
		if err != nil || len(jobs) == 0 {
			continue
		}
		pipe := d.rdb.Pipeline()
		for _, job := range jobs {
			pipe.ZRem(d.ctx, redisDelayedKey, job)
			pipe.LPush(d.ctx, redisQueueKey, []byte(job))
		}
		pipe.Exec(d.ctx) //nolint:errcheck
	}
}
