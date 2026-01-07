package db

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	Username      string `gorm:"uniqueIndex;not null"`
	Email         string `gorm:"uniqueIndex;not null"`
	PasswordHash  string
	Avatar        *string
	GoogleOauthID *string
	TwoFAToken    *string
}

type Friend struct {
	UserID   uint `gorm:"primaryKey;not null"`
	FriendID uint `gorm:"primaryKey;not null"`

	User   User `gorm:"foreignKey:UserID;references:ID"`
	Friend User `gorm:"foreignKey:FriendID;references:ID"`
}

type Token struct {
	gorm.Model

	UserID uint   `gorm:"not null;index"`
	Token  string `gorm:"uniqueIndex;not null"`

	User User `gorm:"foreignKey:UserID;references:ID"`
}

type HeartBeat struct {
	gorm.Model

	UserID uint `gorm:"not null"`
}
