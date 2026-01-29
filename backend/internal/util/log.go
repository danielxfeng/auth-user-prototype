package util

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

func GetLogger(level slog.Leveler) *slog.Logger {
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
		AddSource:  true,
	}))

	slog.SetDefault(logger)
	return logger
}
