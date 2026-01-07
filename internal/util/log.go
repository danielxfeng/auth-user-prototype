package util

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

var Logger *slog.Logger

func InitLogger(level slog.Leveler) {
	Logger = slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
		AddSource:  true,
	}))

	slog.SetDefault(Logger)
}
