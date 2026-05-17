package handler

import (
	"net/http"
	"time"

	"github.com/Takapon2512/recipe-recommend/backend/internal/cognito"
	"github.com/Takapon2512/recipe-recommend/backend/internal/middleware"
	"github.com/Takapon2512/recipe-recommend/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type MeHandler struct {
	userService *service.UserService
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
	claims, ok := c.MustGet(middleware.ContextKeyClaims).(*cognito.Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "claims取得失敗"},
		})
		return
	}

	user, err := h.userService.GetOrCreate(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "ユーザー取得失敗"},
		})
		return
	}

	c.JSON(
		http.StatusOK,
		gin.H{
			"data": MeResponse{
				ID:          user.ID,
				Email:       user.Email,
				DisplayName: user.DisplayName,
				Provider:    user.Provider,
			},
		},
	)
}
