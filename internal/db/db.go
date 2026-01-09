package db

import (
	"context"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/paularynty/transcendence/auth-service-go/internal/util"
)

var DB *gorm.DB

func ConnectDB(dbName string) {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbName), &gorm.Config{TranslateError: true})

	if err != nil {
		panic("failed to connect to db: " + dbName)
	}

	DB.Exec("PRAGMA foreign_keys = ON")

	for _, model := range []any{
		&User{},
		&Friend{},
		&Token{},
		&HeartBeat{},
	} {
		if err := DB.AutoMigrate(model); err != nil {
			panic("failed to migrate model: " + err.Error())
		}
	}

	util.Logger.Info("connected to db")
}

func CloseDB() {
	sqlDB, err := DB.DB()
	if err != nil {
		util.Logger.Error("failed to get db instance", "err", err)
		return
	}

	if err := sqlDB.Close(); err != nil {
		util.Logger.Error("failed to close db", "err", err)
		return
	}

	util.Logger.Info("db connection closed")
}

func ResetDB() {
	util.Logger.Warn("resetting db...")

	ctx := context.Background()
	tables := []string{
		"heart_beats",
		"tokens",
		"friends",
		"users",
	}

	for _, table := range tables {
		err := gorm.G[any](DB).Exec(ctx, "DELETE FROM "+table)
		if err != nil {
			util.Logger.Error("failed to reset table", table, err.Error())
		}
	}

	util.Logger.Info("db is reset")
}
