package handler

import (
	"net/http"
	"time"

	"github.com/Takapon2512/recipe-recommend/backend/internal/cognito"
	"github.com/Takapon2512/recipe-recommend/backend/internal/middleware"
	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"github.com/Takapon2512/recipe-recommend/backend/internal/repository"
	"github.com/Takapon2512/recipe-recommend/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type MeHandler struct {
	userService    *service.UserService
	userRepository *repository.UserRepository
}

type PatchMeRequest struct {
	DisplayName *string `json:"display_name" binding:"omitempty,max=100"`
}

func toMeResponse(u *model.User) MeResponse {
	return MeResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Provider:    u.Provider,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func NewMeHandler(userService *service.UserService) *MeHandler {
	return &MeHandler{userService: userService}
}

type MeResponse struct {
	ID          uint64    `json:"id"`
	Email       string    `json:"email"`
	DisplayName *string   `json:"display_name"`
	Provider    string    `json:"provider"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Getはユーザー情報取得（もしくは新規作成）
func (h *MeHandler) Get(c *gin.Context) {
	_, user, ok := resolveUserWithJIT(c, h.userService)
	if !ok {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toMeResponse(user)})
}

// PATCH /api/me
func (h *MeHandler) Patch(c *gin.Context) {
	_, user, ok := resolveExistingUser(c, h.userRepository)
	if !ok {
		return
	}

	var req PatchMeRequest

	// バリデーションチェック
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()},
		})
		return
	}

	// 更新処理
	updated, err := h.userService.UpdateDisplayName(user, req.DisplayName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "プロフィール更新失敗"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": toMeResponse(updated)})
}

// DELETE /api/me
func (h *MeHandler) Delete(c *gin.Context) {
	_, user, ok := resolveExistingUser(c, h.userRepository)
	if !ok {
		return
	}

	if err := h.userService.Delete(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "アカウント削除失敗"},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// resolveUserWithJIT は claims の取得と GetOrCreate を共通化するヘルパー。
func resolveUserWithJIT(c *gin.Context, svc *service.UserService) (*cognito.Claims, *model.User, bool) {
	claims, ok := c.MustGet(middleware.ContextKeyClaims).(*cognito.Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "claims取得失敗"},
		})
		return nil, nil, false
	}

	user, err := svc.GetOrCreate(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "ユーザー取得失敗"},
		})
		return nil, nil, false
	}

	return claims, user, true
}

// PATCH・DELETE 専用 — Find のみ、見つからなければ 404
func resolveExistingUser(c *gin.Context, repo *repository.UserRepository) (*cognito.Claims, *model.User, bool) {
	claims, ok := c.MustGet(middleware.ContextKeyClaims).(*cognito.Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "claims取得失敗"},
		})
		return nil, nil, false
	}

	user, err := repo.FindByCognitoSub(claims.Sub)
	if err != nil {
		if repository.IsNotFound(err) {
			// 論理削除済み or 未登録（通常ありえないが念のため）
			c.JSON(http.StatusNotFound, gin.H{
				"error": gin.H{"code": "NOT_FOUND", "message": "ユーザーが見つかりません"},
			})
			return nil, nil, false
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "ユーザー取得失敗"},
		})
		return nil, nil, false
	}

	return claims, user, true
}
