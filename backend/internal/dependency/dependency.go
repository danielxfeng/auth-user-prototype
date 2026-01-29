package dependency

import ( 
	"gorm.io/gorm"
	"log/slog"
	"github.com/paularynty/transcendence/auth-service-go/internal/config"
	"github.com/redis/go-redis/v9"
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
