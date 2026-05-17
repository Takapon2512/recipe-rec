package repository

import (
	"fmt"

	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"gorm.io/gorm"
)

type CategoryRepository struct {
	db *gorm.DB
}

func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// FindAll は deleted_at IS NULL のカテゴリを全件返す。
// typeFilter が空文字の場合はフィルタなし。
func (r *CategoryRepository) FindAll(typeFilter string) ([]model.Category, error) {
	var categories []model.Category

	q := r.db.Order("id ASC")
	if typeFilter != "" {
		q = q.Where("type = ?", typeFilter)
	}

	if err := q.Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("categories 取得失敗: %w", err)
	}

	return categories, nil
}
