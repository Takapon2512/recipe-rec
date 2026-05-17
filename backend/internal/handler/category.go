package handler

import (
	"log/slog"
	"net/http"

	"github.com/Takapon2512/recipe-recommend/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
}

func NewCategoryHandler(categoryService *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

// List は GET /api/categories のハンドラ。
func (h *CategoryHandler) List(c *gin.Context) {
	typeFilter := c.Query("type")

	// type クエリパラメータが指定された場合はバリデーション
	if typeFilter != "" && !service.IsValidCategoryType(typeFilter) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_QUERY",
				"message": "type must be one of: food, seasoning, daily",
			},
		})
		return
	}

	categories, err := h.categoryService.ListCategories(typeFilter)
	if err != nil {
		slog.Error("カテゴリ一覧取得失敗", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "internal server error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": categories})
}
