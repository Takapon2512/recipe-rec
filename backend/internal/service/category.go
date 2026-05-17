package service

import (
	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"github.com/Takapon2512/recipe-recommend/backend/internal/repository"
)

type CategoryService struct {
	repo *repository.CategoryRepository
}

// 許容する type 値
var validCategoryTypes = map[string]struct{}{
	"food":      {},
	"seasoning": {},
	"daily":     {},
}

func NewCategoryService(repo *repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

// IsValidCategoryType は type クエリパラメータの値を検証する。
func IsValidCategoryType(t string) bool {
	_, ok := validCategoryTypes[t]
	return ok
}

// ListCategories はカテゴリ一覧を返す。typeFilter が空文字の場合は全件。
func (s *CategoryService) ListCategories(typeFilter string) ([]model.Category, error) {
	return s.repo.FindAll(typeFilter)
}
