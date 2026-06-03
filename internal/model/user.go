package model

import "time"

// internal/model/user.go
type User struct {
	ID       uint   `gorm:"primaryKey"`
	Username string `gorm:"type:varchar(64);not null;uniqueIndex:uk_username"`
	Email    string `gorm:"type:varchar(128);not null;uniqueIndex:uk_email"`
	Password string `gorm:"type:varchar(128);not null"` // bcrypt hash,不是明文

	Status    int8 `gorm:"type:tinyint;not null;default:1"` // 1正常 2禁用
	CreatedAt time.Time
	UpdatedAt time.Time
}
