package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

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
		respondValidationError(c, bindingErrorMessage(err))
		return
	}

	// storage_location バリデーション
	if params.StorageLocation != "" && !model.ValidStorageLocations[params.StorageLocation] {
		respondValidationError(c, "保管場所 は次のいずれかを指定してください: fridge, freezer, pantry")
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
		respondValidationError(c, bindingErrorMessage(err))
		return
	}

	item, err := h.inventoryService.Create(user.ID, req)
	if err != nil {
		var ve *service.ValidationError
		switch {
		case errors.Is(err, service.ErrCategoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "specified category_id does not exist",
				},
			})
		case errors.As(err, &ve):
			slog.Warn("inventory create validation failed", "error", err)
			respondValidationError(c, ve.Error())
		default:
			// DB エラー等は 500
			slog.Error("inventory create failed", "error", err)
			respondInternalError(c)
		}
		return
	}

	resp := toInventoryResponse(item)
	c.JSON(http.StatusCreated, gin.H{"data": resp})
}

// GetByID は GET /api/inventory/:id のハンドラ。
func (h *InventoryHandler) GetByID(c *gin.Context) {
	_, user, ok := resolveExistingUser(c, h.userService)
	if !ok {
		return
	}

	id, ok := parseInventoryID(c)
	if !ok {
		return
	}

	item, err := h.inventoryService.GetInventoryItem(user.ID, id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondNotFound(c)
			return
		}
		slog.Error("inventory get failed", "id", id, "error", err)
		respondInternalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toInventoryResponse(item)})
}

// Update は PATCH /api/inventory/:id のハンドラ。
// 送信されたフィールドのみ更新する部分更新。
// category_id に null を明示送信するとカテゴリを未分類に変更する。
func (h *InventoryHandler) Update(c *gin.Context) {
	_, user, ok := resolveExistingUser(c, h.userService)
	if !ok {
		return
	}

	id, ok := parseInventoryID(c)
	if !ok {
		return
	}

	// --- リクエストボディのパース ---
	// ShouldBindJSON だけでは "category_id": null（明示的 null）と
	// フィールド未送信を区別できないため、map で一度受けて判定する。
	var raw map[string]json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		respondValidationError(c, bindingErrorMessage(err))
		return
	}

	var req model.UpdateInventoryItemRequest

	// 各フィールドを手動でデコード（送信されたキーのみポインタにセット）
	if v, exists := raw["name"]; exists {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			respondValidationError(c, "name は文字列で指定してください")
			return
		}
		req.Name = &s
	}
	if v, exists := raw["category_id"]; exists {
		if string(v) == "null" {
			// null 明示 → カテゴリ解除
			req.ClearCategoryID = true
		} else {
			var categoryID int
			if err := json.Unmarshal(v, &categoryID); err != nil {
				respondValidationError(c, "category_id は整数で指定してください")
				return
			}
			req.CategoryID = &categoryID
		}
	}
	if v, exists := raw["quantity"]; exists {
		var f float64
		if err := json.Unmarshal(v, &f); err != nil {
			respondValidationError(c, "quantity は数値で指定してください")
			return
		}
		req.Quantity = &f
	}
	if v, exists := raw["unit"]; exists {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			respondValidationError(c, "unit は文字列で指定してください")
			return
		}
		req.Unit = &s
	}
	if v, exists := raw["purchased_at"]; exists {
		if string(v) == "null" {
			// null 明示 → フィールドをクリア（空文字列で parseDate に nil を返させる）
			empty := ""
			req.PurchasedAt = &empty
		} else {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				respondValidationError(c, "purchased_at は文字列で指定してください")
				return
			}
			req.PurchasedAt = &s
		}
	}
	if v, exists := raw["expires_at"]; exists {
		if string(v) == "null" {
			empty := ""
			req.ExpiresAt = &empty
		} else {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				respondValidationError(c, "expires_at は文字列で指定してください")
				return
			}
			req.ExpiresAt = &s
		}
	}
	if v, exists := raw["storage_location"]; exists {
		if string(v) == "null" {
			empty := ""
			req.StorageLocation = &empty
		} else {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				respondValidationError(c, "storage_location は文字列で指定してください")
				return
			}
			req.StorageLocation = &s
		}
	}
	if v, exists := raw["memo"]; exists {
		if string(v) == "null" {
			empty := ""
			req.Memo = &empty
		} else {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				respondValidationError(c, "memo は文字列で指定してください")
				return
			}
			req.Memo = &s
		}
	}
	item, err := h.inventoryService.UpdateInventoryItem(user.ID, id, &req)
	if err != nil {
		var ve *service.ValidationError
		switch {
		case errors.Is(err, service.ErrNotFound):
			respondNotFound(c)
		case errors.Is(err, service.ErrCategoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "specified category_id does not exist",
				},
			})
		case errors.As(err, &ve):
			slog.Warn("inventory update validation failed", "id", id, "error", err)
			respondValidationError(c, ve.Error())
		default:
			slog.Error("inventory update failed", "id", id, "error", err)
			respondInternalError(c)
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toInventoryResponse(item)})
}

// Delete は DELETE /api/inventory/:id のハンドラ。論理削除、204 No Content を返す。
func (h *InventoryHandler) Delete(c *gin.Context) {
	_, user, ok := resolveExistingUser(c, h.userService)
	if !ok {
		return
	}

	id, ok := parseInventoryID(c)
	if !ok {
		return
	}

	if err := h.inventoryService.DeleteInventoryItem(user.ID, id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondNotFound(c)
			return
		}
		slog.Error("inventory delete failed", "id", id, "error", err)
		respondInternalError(c)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetExpiring は GET /api/inventory/expiring のハンドラ。
func (h *InventoryHandler) GetExpiring(c *gin.Context) {
	_, user, ok := resolveExistingUser(c, h.userService)
	if !ok {
		return
	}

	var params model.ExpiringParams
	if err := c.ShouldBindQuery(&params); err != nil {
		respondValidationError(c, bindingErrorMessage(err))
		return
	}

	items, err := h.inventoryService.GetExpiring(user.ID, params)
	if err != nil {
		slog.Error("inventory expiring failed", "error", err)
		respondInternalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

// Suggest は GET /api/inventory/suggest のハンドラ。
func (h *InventoryHandler) Suggest(c *gin.Context) {
	_, user, ok := resolveExistingUser(c, h.userService)
	if !ok {
		return
	}

	var params model.SuggestParams
	if err := c.ShouldBindQuery(&params); err != nil {
		respondValidationError(c, bindingErrorMessage(err))
		return
	}

	items, err := h.inventoryService.Suggest(user.ID, params)
	if err != nil {
		slog.Error("inventory suggest failed", "error", err)
		respondInternalError(c)
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// --- ヘルパー ---

// parseInventoryID はパスパラメータ :id を uint64 にパースする。
// パース失敗時は 400 を返し false を返す。
func parseInventoryID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		respondValidationError(c, "id は1以上の整数で指定してください")
		return 0, false
	}
	return id, true
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

func respondNotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{
		"error": gin.H{"code": "NOT_FOUND", "message": "not found"},
	})
}

func respondInternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": gin.H{"code": "INTERNAL_ERROR", "message": "internal server error"},
	})
}
