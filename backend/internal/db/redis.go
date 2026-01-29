package db

import (
	"context"
	"log/slog"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/redis/go-redis/v9"
)

func GetRedis(redisURL string, cfg *config.Config, logger *slog.Logger) *redis.Client {
	if !cfg.IsRedisEnabled {
		logger.Info("redis is disabled by config")
		return nil
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

	logger.Info("connected to redis")

	return client
}

func CloseRedis(client *redis.Client, logger *slog.Logger) {
	if client == nil {
		return
	}

	err := client.Close()
	if err != nil {
		logger.Error("failed to close redis connection", "error", err)
	} else {
		logger.Info("redis connection closed")
	}
}
