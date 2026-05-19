package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"github.com/Takapon2512/recipe-recommend/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type InventoryHandler struct {
	inventoryService *service.InventoryService
	userService      *service.UserService
}

func NewInventoryHandler(inventoryService *service.InventoryService, userService *service.UserService) *InventoryHandler {
	return &InventoryHandler{inventoryService: inventoryService, userService: userService}
}

// List は GET /api/inventory のハンドラ。
func (h *InventoryHandler) List(c *gin.Context) {
	_, user, ok := resolveExistingUser(c, h.userService)
	if !ok {
		return
	}

	var params model.ListInventoryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	// storage_location バリデーション
	if params.StorageLocation != "" && !model.ValidStorageLocations[params.StorageLocation] {
		respondValidationError(c, "storage_location must be one of: fridge, freezer, pantry")
		return
	}

	result, err := h.inventoryService.List(user.ID, params)
	if err != nil {
		slog.Error("inventory list failed", "error", err)
		respondInternalError(c)
		return
	}

	// レスポンス変換
	responses := make([]model.InventoryResponse, len(result.Items))
	for i, item := range result.Items {
		responses[i] = toInventoryResponse(&item)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       responses,
		"pagination": result.Pagination,
	})
}

// Create は POST /api/inventory のハンドラ。
func (h *InventoryHandler) Create(c *gin.Context) {
	_, user, ok := resolveExistingUser(c, h.userService)
	if !ok {
		return
	}

	var req model.CreateInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, err.Error())
		return
	}

	item, err := h.inventoryService.Create(user.ID, req)
	if err != nil {
		if errors.Is(err, service.ErrCategoryNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "specified category_id does not exist",
				},
			})
			return
		}
		// unit / storage_location / date のバリデーションエラー
		slog.Warn("inventory create validation failed", "error", err)
		respondValidationError(c, err.Error())
		return
	}

	resp := toInventoryResponse(item)
	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// レスポンス成形
// toInventoryResponse は InventoryItem を InventoryResponse に変換する。
func toInventoryResponse(item *model.InventoryItem) model.InventoryResponse {
	resp := model.InventoryResponse{
		ID:              item.ID,
		Name:            item.Name,
		Quantity:        item.Quantity,
		Unit:            item.Unit,
		StorageLocation: item.StorageLocation,
		Memo:            item.Memo,
		CreatedAt:       item.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       item.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}

	if item.Category != nil {
		resp.Category = &model.CategoryResponse{
			ID:   item.Category.ID,
			Name: item.Category.Name,
		}
	}

	if item.PurchasedAt != nil {
		s := item.PurchasedAt.Format("2006-01-02")
		resp.PurchasedAt = &s
	}
	if item.ExpiresAt != nil {
		s := item.ExpiresAt.Format("2006-01-02")
		resp.ExpiresAt = &s
	}

	return resp
}

func respondValidationError(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error": gin.H{"code": "VALIDATION_ERROR", "message": message},
	})
}

func respondInternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{"code": "INTERNAL_ERROR", "message": "internal server error"},
	})
}
