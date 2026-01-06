package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/paularynty/transcendence/auth-service-go/configs"
	"github.com/paularynty/transcendence/auth-service-go/internal/users"

	"log/slog"

	sloggin "github.com/samber/slog-gin"
)

func SetupRouter(logger *slog.Logger) *gin.Engine {
	r := gin.New()

	config := sloggin.Config{
		DefaultLevel:     slog.LevelInfo,
		ClientErrorLevel: slog.LevelWarn,
		ServerErrorLevel: slog.LevelError,
	}

	r.Use(gin.Recovery())
	r.Use(sloggin.NewWithConfig(logger, config))

	return r
}

func main() {
	// logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// config
	godotenv.Load()
	cfg := configs.LoadConfig()

	// router
	r := SetupRouter(logger)
	users.UsersRouter(r.Group("/api/user"), logger, cfg)

	r.GET("/api/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.Run(":3003")
}
