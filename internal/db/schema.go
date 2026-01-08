package db

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	Username      string `gorm:"uniqueIndex;not null"`
	Email         string `gorm:"uniqueIndex;not null"`
	PasswordHash  *string
	Avatar        *string
	GoogleOauthID *string `gorm:"uniqueIndex"`
	TwoFAToken    *string
}

type Friend struct {
	UserID   uint `gorm:"primaryKey;not null"`
	FriendID uint `gorm:"primaryKey;not null"`

	User   User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Friend User `gorm:"foreignKey:FriendID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Token struct {
	gorm.Model

	UserID uint   `gorm:"not null;index"`
	Token  string `gorm:"uniqueIndex;not null"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type HeartBeat struct {
	gorm.Model

	UserID     uint      `gorm:"uniqueIndex;not null"`
	LastSeenAt time.Time `gorm:"not null"`

	User User `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}