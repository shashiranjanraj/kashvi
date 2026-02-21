package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shashiranjanraj/kashvi/config"
)

var RDB *redis.Client
var Ctx = context.Background()

// Connect initialises the Redis client and verifies the connection with a ping.
// Returns an error so the caller can react (log warning, fall back, or abort).
func Connect() error {
	RDB = redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr(),
		Password: config.RedisPassword(),
		DB:       0,
	})

	if err := RDB.Ping(Ctx).Err(); err != nil {
		RDB = nil // mark as unavailable so Get/Set/Del no-op safely
		return fmt.Errorf("cache: redis ping: %w", err)
	}
	return nil
}

// Get retrieves a cached value by key and unmarshals into dest.
// Returns true on a cache hit, false on miss or error.
func Get(key string, dest interface{}) bool {
	if RDB == nil {
		return false
	}

	val, err := RDB.Get(Ctx, key).Result()
	if err != nil {
		return false
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false
	}

	return true
}

// Set stores value in Redis under key for the given TTL.
func Set(key string, value interface{}, ttl time.Duration) error {
	if RDB == nil {
		return nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return RDB.Set(Ctx, key, data, ttl).Err()
}

// Del removes one or more keys from Redis.
func Del(keys ...string) error {
	if RDB == nil {
		return nil
	}
	return RDB.Del(Ctx, keys...).Err()
}

// Forget is an alias for Del (Laravel-style).
func Forget(key string) error {
	return Del(key)
}
