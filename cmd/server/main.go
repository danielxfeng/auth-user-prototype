package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"log/slog"

	sloggin "github.com/samber/slog-gin"
)

func main() {
  
  logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
  slog.SetDefault(logger)
  config := sloggin.Config {
    DefaultLevel:     slog.LevelInfo,
    ClientErrorLevel: slog.LevelWarn,
    ServerErrorLevel: slog.LevelError,
  }

  r := gin.New()

  r.Use(sloggin.NewWithConfig(logger, config))
  r.Use(gin.Recovery())
  
  r.GET("/api/ping", func(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
      "message": "pong",
    })
  })

  r.Run(":3003")
}