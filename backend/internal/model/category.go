package model

import (
	"time"

	"gorm.io/gorm"
)

type Category struct {
	ID                     int            `gorm:"primaryKey;autoIncrement"           json:"id"`
	Name                   string         `gorm:"column:name;not null;size:50"       json:"name"`
	Type                   string         `gorm:"column:type;not null;size:20"       json:"type"`
	DefaultStorageLocation *string        `gorm:"column:default_storage_location"    json:"default_storage_location"`
	CreatedAt              time.Time      `gorm:"column:created_at;not null;autoCreateTime" json:"-"`
	UpdatedAt              time.Time      `gorm:"column:updated_at;not null;autoUpdateTime" json:"-"`
	DeletedAt              gorm.DeletedAt `gorm:"column:deleted_at;index"            json:"-"`
}

func (Category) TableName() string {
	return "categories"
}
