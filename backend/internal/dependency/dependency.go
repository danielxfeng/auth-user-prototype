package dependency

import (
	"log/slog"

	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/paularynty/transcendence/auth-service-go/internal/db"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Dependency struct {
	Cfg    *config.Config
	DB     *gorm.DB
	Redis  *redis.Client
	Logger *slog.Logger
}

func NewDependency(cfg *config.Config, db *gorm.DB, redis *redis.Client, logger *slog.Logger) *Dependency {
	return &Dependency{
		Cfg:    cfg,
		DB:     db,
		Redis:  redis,
		Logger: logger,
	}
}

func InitDependency(logger *slog.Logger) (*Dependency, error) {
	cfg, err := config.LoadConfigFromEnv()
	if err != nil {
		return nil, err
	}

	myDB, err := db.GetDB(cfg.DbAddress, logger)
	if err != nil {
		return nil, err
	}

	redis, err := db.GetRedis(cfg.RedisURL, cfg, logger)
	if err != nil {
		return nil, err
	}

	return NewDependency(cfg, myDB, redis, logger), nil
}

func CloseDependency(dep *Dependency) {
	db.CloseDB(dep.DB, dep.Logger)
	db.CloseRedis(dep.Redis, dep.Logger)
}
