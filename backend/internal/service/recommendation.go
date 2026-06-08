package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Takapon2512/recipe-recommend/backend/internal/anthropic"
	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"github.com/Takapon2512/recipe-recommend/backend/internal/repository"
	"gorm.io/gorm"
)

// ----------------------------------------------------------------
// 定数
// ----------------------------------------------------------------

const (
	// rateLimitWindow はレート制限の集計ウィンドウ（1時間）。
	rateLimitWindow = 1 * time.Hour

	// rateLimitMax は 1ユーザー・1時間あたりの上限リクエスト数。
	rateLimitMax = 10

	// defaultCount はリクエストで count が未指定（0）のときの提案件数。
	defaultCount = 3

	// llmTimeout は Claude API 呼び出しのタイムアウト。
	llmTimeout = 60 * time.Second

	// jobIDPrefix は job_id の先頭に付けるプレフィックス。
	jobIDPrefix = "rec_"
)

// ----------------------------------------------------------------
// エラー定義
// ----------------------------------------------------------------

// ErrRateLimited はレート制限超過を表す番哨エラー。
// handler 層で 429 に変換する。
var ErrRateLimited = errors.New("rate limit exceeded")

// ----------------------------------------------------------------
// インターフェース
// ----------------------------------------------------------------

// RecommendationService は recommendation のビジネスロジックを提供する。
type RecommendationService interface {
	// EnqueueRecommendation は recommendation ジョブを DB に登録し、
	// job_id・ステータス・作成日時を返す。
	// レート制限超過時は ErrRateLimited を返す。
	EnqueueRecommendation(ctx context.Context, userID uint64, req *model.CreateRecommendationRequest) (*model.CreateRecommendationResponse, error)

	// ProcessJob は pending 状態のジョブを1件取得して LLM を呼び出し、
	// 結果を DB に保存する。対象ジョブがなければ何もしない。
	// ワーカーのポーリングループから定期的に呼ばれることを想定する。
	ProcessJob(ctx context.Context) error

	// GetJob は job_id でジョブを取得して返す。
	// 存在しない・他ユーザー所有の場合は ErrNotFound を返す。
	GetJob(ctx context.Context, userID uint64, jobID string) (*model.GetJobResponse, error)
}

// ----------------------------------------------------------------
// 実装
// ----------------------------------------------------------------

type recommendationService struct {
	recRepo repository.RecommendationRepository
	invRepo *repository.InventoryRepository
	llm     anthropic.Client
}

// NewRecommendationService は RecommendationService のコンストラクタ。
func NewRecommendationService(
	recRepo repository.RecommendationRepository,
	invRepo *repository.InventoryRepository,
	llm anthropic.Client,
) RecommendationService {
	return &recommendationService{
		recRepo: recRepo,
		invRepo: invRepo,
		llm:     llm,
	}
}

// ----------------------------------------------------------------
// EnqueueRecommendation
// ----------------------------------------------------------------

func (s *recommendationService) EnqueueRecommendation(
	ctx context.Context,
	userID uint64,
	req *model.CreateRecommendationRequest,
) (*model.CreateRecommendationResponse, error) {
	// --- 1. count のデフォルト補完 ---
	if req.Count == 0 {
		req.Count = defaultCount
	}

	// --- 2. レート制限チェック ---
	since := time.Now().Add(-rateLimitWindow)
	count, err := s.recRepo.CountRecentByUser(ctx, userID, since)
	if err != nil {
		return nil, fmt.Errorf("EnqueueRecommendation CountRecentByUser: %w", err)
	}
	if count >= rateLimitMax {
		return nil, ErrRateLimited
	}

	// --- 3. request_payload を JSON シリアライズ ---
	payloadBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("EnqueueRecommendation marshal request: %w", err)
	}

	// --- 4. DB に pending ジョブを INSERT ---
	log := &model.RecommendationLog{
		UserID:         userID,
		Status:         model.RecommendationStatusPending,
		RequestPayload: payloadBytes,
	}
	if err := s.recRepo.Create(ctx, log); err != nil {
		return nil, fmt.Errorf("EnqueueRecommendation Create: %w", err)
	}

	return &model.CreateRecommendationResponse{
		JobID:     jobIDPrefix + fmt.Sprintf("%d", log.ID),
		Status:    string(model.RecommendationStatusPending),
		CreatedAt: log.CreatedAt,
	}, nil
}

// ----------------------------------------------------------------
// ProcessJob
// ----------------------------------------------------------------
func (s *recommendationService) ProcessJob(ctx context.Context) error {
	// --- 1. pending ジョブを1件取得 ---
	job, err := s.recRepo.FindPendingJob(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// pending ジョブなし: 正常
			return nil
		}
		return fmt.Errorf("ProcessJob FindPendingJob: %w", err)
	}

	logger := slog.With("job_id", job.ID, "user_id", job.UserID)
	logger.InfoContext(ctx, "processing recommendation job")

	// --- 2. statusをRunningに更新 ---
	if err := s.recRepo.UpdateStatus(ctx, job.ID, model.RecommendationStatusRunning); err != nil {
		return fmt.Errorf("ProcessJob UpdateStatus running: %w", err)
	}

	// --- 3. リクエストペイロードをデシリアライズ ---
	var req model.CreateRecommendationRequest
	if err := json.Unmarshal(job.RequestPayload, &req); err != nil {
		saveErr := s.recRepo.SaveError(ctx, job.ID, "PAYLOAD_ERROR", "failed to parse request payload")
		return fmt.Errorf("ProcessJob unmarshal payload: %w (saveErr: %v)", err, saveErr)
	}

	// --- 4. 在庫を取得してプロンプト用データを組み立て ---
	items, err := s.invRepo.FindActiveByUser(job.UserID, req.Filters.PreferExpiring)
	if err != nil {
		saveErr := s.recRepo.SaveError(ctx, job.ID, "INVENTORY_ERROR", "failed to fetch inventory")
		return fmt.Errorf("ProcessJob FindActiveByUser: %w (saveErr: %v)", err, saveErr)
	}

	llmItems := toAnthropicItems(items)

	// --- 5. LLM 呼び出し（タイムアウト付き） ---
	llmCtx, cancel := context.WithTimeout(ctx, llmTimeout)
	defer cancel()

	start := time.Now()
	result, err := s.llm.Recommend(llmCtx, &anthropic.RecommendRequest{
		InventoryItems: llmItems,
		Filters:        req.Filters,
		Count:          req.Count,
	})
	latencyMs := int(time.Since(start).Milliseconds())

	if err != nil {
		errCode, errMsg := classifyLLMError(err)
		logger.WarnContext(ctx, "LLM call failed",
			"error", err,
			"code", errCode,
			"latency_ms", latencyMs,
		)
		saveErr := s.recRepo.SaveError(ctx, job.ID, errCode, errMsg)
		if saveErr != nil {
			return fmt.Errorf("ProcessJob SaveError: %w (original: %v)", saveErr, err)
		}
		return nil // ジョブ失敗は正常終了として扱い、ワーカーを止めない
	}

	// --- 6. 結果を保存して succeeded に更新 ---
	if err := s.recRepo.SaveResult(ctx, job.ID, result, latencyMs); err != nil {
		return fmt.Errorf("ProcessJob SaveResult: %w", err)
	}

	logger.InfoContext(ctx, "recommendation job succeeded",
		"recipe_count", len(result.Recipes),
		"latency_ms", latencyMs,
	)
	return nil
}

// ----------------------------------------------------------------
// GetJob
// ----------------------------------------------------------------

func (s *recommendationService) GetJob(ctx context.Context, userID uint64, jobID string) (*model.GetJobResponse, error) {
	if !strings.HasPrefix(jobID, jobIDPrefix) {
		return nil, ErrNotFound
	}
	rawID := strings.TrimPrefix(jobID, jobIDPrefix)
	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || id == 0 {
		return nil, ErrNotFound
	}

	log, err := s.recRepo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetJob FindByID: %w", err)
	}

	if log.UserID != userID {
		return nil, ErrNotFound
	}

	resp := &model.GetJobResponse{
		JobID:     jobID,
		Status:    string(log.Status),
		CreatedAt: log.CreatedAt,
		UpdatedAt: log.UpdatedAt,
	}

	if log.Status == model.RecommendationStatusSucceeded && log.ResponsePayload != nil {
		var result model.RecommendationResultPayload
		if err := json.Unmarshal(log.ResponsePayload, &result); err != nil {
			return nil, fmt.Errorf("GetJob unmarshal response: %w", err)
		}
		resp.Result = &result
	}

	if log.Status == model.RecommendationStatusFailed && log.ErrorCode != nil {
		resp.Error = &model.JobError{Code: *log.ErrorCode}
		if log.ErrorMessage != nil {
			resp.Error.Message = *log.ErrorMessage
		}
	}

	return resp, nil
}

// ----------------------------------------------------------------
// ヘルパー
// ----------------------------------------------------------------
// toAnthropicItems は repository.RecommendationInventoryItem を anthropic.InventoryItem に変換する。
func toAnthropicItems(items []repository.RecommendationInventoryItem) []anthropic.InventoryItem {
	result := make([]anthropic.InventoryItem, 0, len(items))
	for _, item := range items {
		ai := anthropic.InventoryItem{
			Name:     item.Name,
			Quantity: item.Quantity,
			Unit:     item.Unit,
		}
		if item.ExpiresAt != nil {
			s := item.ExpiresAt.Format("2006-01-02")
			ai.ExpiresAt = &s
		}
		result = append(result, ai)
	}
	return result
}

// classifyLLMError は LLM 呼び出しエラーをエラーコードとメッセージに変換する。
func classifyLLMError(err error) (code, message string) {
	if errors.Is(err, context.DeadlineExceeded) {
		return "LLM_TIMEOUT", "Claude API did not respond within 60 seconds"
	}
	return "LLM_ERROR", fmt.Sprintf("Claude API error: %s", err.Error())
}
