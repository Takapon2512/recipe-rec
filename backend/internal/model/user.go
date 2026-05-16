package model

import (
	"time"
)

type User struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement"`
	CognitoSub  string     `gorm:"column:cognito_sub;uniqueIndex;not null;size:64"`
	Email       string     `gorm:"column:email;index;not null;size:255"`
	DisplayName *string    `gorm:"column:display_name;size:100"`
	Provider    string     `gorm:"column:provider;not null;size:20"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;not null;autoUpdateTime"`
	DeletedAt   *time.Time `gorm:"column:deleted_at;index"`
}

func (User) TableName() string {
	return "users"
}
