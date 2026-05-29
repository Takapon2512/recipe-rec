package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"github.com/Takapon2512/recipe-recommend/backend/internal/service"
	"github.com/gin-gonic/gin"
)

type RecommendationHandler struct {
	svc         service.RecommendationService
	userService *service.UserService
}

func NewRecommendationHandler(svc service.RecommendationService, userService *service.UserService) *RecommendationHandler {
	return &RecommendationHandler{svc: svc, userService: userService}
}

// Create は POST /api/recommendations のハンドラ。202 Accepted でジョブ情報を返す。
func (h *RecommendationHandler) Create(c *gin.Context) {
	_, user, ok := resolveExistingUser(c, h.userService)
	if !ok {
		return
	}

	var req model.CreateRecommendationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondValidationError(c, bindingErrorMessage(err))
		return
	}

	if req.Count != 0 && (req.Count < 1 || req.Count > 5) {
		respondValidationError(c, "count は 1〜5 の範囲で指定してください")
		return
	}

	if req.Filters.Genre != nil {
		switch *req.Filters.Genre {
		case "japanese", "western", "chinese", "other":
		default:
			respondValidationError(c, "genre は japanese / western / chinese / other のいずれかを指定してください")
			return
		}
	}

	resp, err := h.svc.EnqueueRecommendation(c.Request.Context(), user.ID, &req)
	if err != nil {
		if errors.Is(err, service.ErrRateLimited) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": gin.H{"code": "RATE_LIMITED", "message": "リクエスト上限（10回/時間）を超えました"},
			})
			return
		}
		slog.Error("recommendation enqueue failed", "error", err)
		respondInternalError(c)
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"data": resp})
}

// GetJob は GET /api/recommendations/:job_id のハンドラ。
func (h *RecommendationHandler) GetJob(c *gin.Context) {
	_, user, ok := resolveExistingUser(c, h.userService)
	if !ok {
		return
	}

	jobID := c.Param("job_id")

	resp, err := h.svc.GetJob(c.Request.Context(), user.ID, jobID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondNotFound(c)
			return
		}
		slog.Error("recommendation get job failed", "job_id", jobID, "error", err)
		respondInternalError(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": resp})
}
