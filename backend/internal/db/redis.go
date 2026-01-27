package db

import (
	"context"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"
	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

// ConnectOptionalRedis connects to Redis and sets the global Redis client.
//
// If Redis is disabled via configuration, or if the connection attempt fails,
// Redis remains nil and config.Cfg.IsRedisEnabled is set to false.
func ConnectOptionalRedis(redisURL string) {
	if !config.Cfg.IsRedisEnabled {
		util.Logger.Info("redis is disabled by config")
		return
	}

	opt, err := redis.ParseURL(redisURL)

	if err != nil {
		util.Logger.Error("failed to parse redis url", "err", err)
		config.Cfg.IsRedisEnabled = false
		return
	}

  	client := redis.NewClient(opt)

	ctx := context.Background()
	
	_, err = client.Ping(ctx).Result()
	if err != nil {
		util.Logger.Error("failed to connect to redis", "err", err)
		config.Cfg.IsRedisEnabled = false
		return
	}

	Redis = client

	util.Logger.Info("connected to redis")
}