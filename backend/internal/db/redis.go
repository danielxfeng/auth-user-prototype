package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/redis/go-redis/v9"
)

func GetRedis(redisURL string, cfg *config.Config, logger *slog.Logger) (*redis.Client, error) {
	if !cfg.IsRedisEnabled {
		return nil, nil
	}

	opt, err := redis.ParseURL(redisURL)

	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url, err: %w", err)
	}

	client := redis.NewClient(opt)

	ctx := context.Background()

	_, err = client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("connected to redis")

	return client, nil
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
