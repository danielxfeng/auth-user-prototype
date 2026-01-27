package db

import (
	"context"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/util"
	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

func ConnectRedis(redisURL string) {
	if !config.Cfg.IsRedisEnabled {
		util.Logger.Info("redis is disabled by config")
		return
	}

	opt, err := redis.ParseURL(redisURL)

	if err != nil {
		panic("failed to parse redis url, err: " + err.Error())
	}

  	client := redis.NewClient(opt)

	ctx := context.Background()
	
	_, err = client.Ping(ctx).Result()
	if err != nil {
		panic("failed to connect to redis: " + err.Error())
	}

	Redis = client

	util.Logger.Info("connected to redis")
}

func CloseRedis() {
	if Redis == nil {
		return
	}

	err := Redis.Close()
	if err != nil {
		util.Logger.Error("failed to close redis connection", "error", err)
	} else {
		util.Logger.Info("redis connection closed")
	}
}