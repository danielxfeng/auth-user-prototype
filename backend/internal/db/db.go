package db

import (
	"context"
	"fmt"
	"log/slog"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func GetDB(dbName string, logger *slog.Logger) (*gorm.DB, error) {
	var err error
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{TranslateError: true})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to db: %w", err)
	}

	db.Exec("PRAGMA foreign_keys = ON")

	for _, model := range []any{
		&User{},
		&Friend{},
		&Token{},
		&HeartBeat{},
	} {
		if err := db.AutoMigrate(model); err != nil {
			return nil, fmt.Errorf("failed to migrate model: %w", err)
		}
	}

	logger.Info("connected to db")

	return db, nil
}

func CloseDB(db *gorm.DB, logger *slog.Logger) {
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("failed to get db instance", "err", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		logger.Error("failed to close db", "err", err)
		return
	}

	logger.Info("db connection closed")
}

func ResetDB(db *gorm.DB, logger *slog.Logger) {
	logger.Warn("resetting db...")
	ctx := context.Background()
	tables := []string{
		"heart_beats",
		"tokens",
		"friends",
		"users",
	}

	for _, table := range tables {
		err := gorm.G[any](db).Exec(ctx, "DELETE FROM "+table)
		if err != nil {
			logger.Error("failed to reset table", table, err.Error())
		}
	}

	logger.Info("db is reset")
}
