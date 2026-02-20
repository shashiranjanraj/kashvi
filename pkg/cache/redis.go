package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shashiranjanraj/kashvi/config"
)

var RDB *redis.Client
var Ctx = context.Background()

func Connect() {
	RDB = redis.NewClient(&redis.Options{
		Addr: config.RedisAddr(),
	})
}

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
