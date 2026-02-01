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

// LogFatalErr logs the error message and exits the program if err is not nil.
func LogFatalErr(logger *slog.Logger, err error, msg string) {
	if err != nil {
		logger.Error(msg, "err", err)
		os.Exit(1)
	}
}
