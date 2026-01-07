package service

import (
	"gorm.io/gorm"

	"github.com/paularynty/transcendence/auth-service-go/internal/db"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService() *UserService {
	return &UserService{
		db: db.DB,
	}
}