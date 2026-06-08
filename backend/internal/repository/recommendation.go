package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"gorm.io/gorm"
)

type RecommendationRepository interface {
	// Create は status=pending の新規ジョブを登録する。
	Create(ctx context.Context, log *model.RecommendationLog) error

	// FindByID は job_id（= recommendation_logs.id）で1件取得する。
	// 存在しない・論理削除済みの場合は gorm.ErrRecordNotFound を返す。
	FindByID(ctx context.Context, id uint64) (*model.RecommendationLog, error)

	// FindPendingJob は status='pending' のジョブを登録順に1件取得する。
	// 該当なしの場合は gorm.ErrRecordNotFound を返す。
	FindPendingJob(ctx context.Context) (*model.RecommendationLog, error)

	// UpdateStatus は指定ジョブの status のみを更新する。
	UpdateStatus(ctx context.Context, id uint64, status model.RecommendationStatus) error

	// SaveResult は LLM 応答を保存し、status を succeeded に変更する。
	SaveResult(ctx context.Context, id uint64, payload *model.RecommendationResultPayload, latencyMs int) error

	// SaveError はエラー情報を保存し、status を failed に変更する。
	SaveError(ctx context.Context, id uint64, code, message string) error

	// CountRecentByUser は指定ユーザーの直近 since 以降のジョブ件数を返す。
	// レート制限チェック用（service 層から呼ばれる）。
	CountRecentByUser(ctx context.Context, userID uint64, since time.Time) (int64, error)
}

// recommendationRepository は RecommendationRepository の GORM 実装。
type recommendationRepository struct {
	db *gorm.DB
}
 
// NewRecommendationRepository は RecommendationRepository のコンストラクタ。
func NewRecommendationRepository(db *gorm.DB) RecommendationRepository {
	return &recommendationRepository{db: db}
}


// ----------------------------------------------------------------
// Create
// ----------------------------------------------------------------
func (r *recommendationRepository) Create(ctx context.Context, log *model.RecommendationLog) error {
	result := r.db.WithContext(ctx).Create(log)
	if result.Error != nil {
		return fmt.Errorf("recommendation_logs INSERT: %w", result.Error)
	}
	return nil
}

// ----------------------------------------------------------------
// FindByID
// ----------------------------------------------------------------
func (r *recommendationRepository) FindByID(ctx context.Context, id uint64) (*model.RecommendationLog, error) {
	var log model.RecommendationLog
	result := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&log)
	if result.Error != nil {
		// gorm.ErrRecordNotFound をそのまま返し、呼び出し元で判定させる。
		return nil, fmt.Errorf("recommendation_logs FindByID id=%d: %w", id, result.Error)
	}
	return &log, nil
}

// ----------------------------------------------------------------
// FindPendingJob
// ----------------------------------------------------------------

// FindPendingJob は status='pending' のジョブを created_at 昇順で1件取得する。
// シングルワーカー前提のシンプルな FIFO 実装。
func (r *recommendationRepository) FindPendingJob(ctx context.Context) (*model.RecommendationLog, error) {
	var log model.RecommendationLog
	result := r.db.WithContext(ctx).
		Where("status = ?", model.RecommendationStatusPending).
		Order("created_at ASC").
		First(&log)
	if result.Error != nil {
		return nil, fmt.Errorf("recommendation_logs FindPendingJob: %w", result.Error)
	}
	return &log, nil
}

// ----------------------------------------------------------------
// UpdateStatus
// ----------------------------------------------------------------
func (r *recommendationRepository) UpdateStatus(ctx context.Context, id uint64, status model.RecommendationStatus) error {
	result := r.db.WithContext(ctx).
	Model(&model.RecommendationLog{}).
	Where("id = ?", id).
	Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("recommendation_logs UpdateStatus id=%d status=%s: %w", id, status, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("recommendation_logs UpdateStatus id=%d: %w", id, gorm.ErrRecordNotFound)
	}
	return nil
}

// ----------------------------------------------------------------
// SaveResult
// ----------------------------------------------------------------
func (r *recommendationRepository) SaveResult(
	ctx context.Context,
	id uint64,
	payload *model.RecommendationResultPayload,
	latencyMs int,
) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("recommendation_logs SaveResult marshal: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&model.RecommendationLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":           model.RecommendationStatusSucceeded,
			"response_payload": raw,
			"latency_ms":       latencyMs,
		})
	
	if result.Error != nil {
		return fmt.Errorf("recommendation_logs SaveResult id=%d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("recommendation_logs SaveResult id=%d: %w", id, gorm.ErrRecordNotFound)
	}
	return nil
}

// ----------------------------------------------------------------
// SaveError
// ----------------------------------------------------------------
func (r *recommendationRepository) SaveError(ctx context.Context, id uint64, code, message string) error {
	result := r.db.WithContext(ctx).
		Model(&model.RecommendationLog{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        model.RecommendationStatusFailed,
			"error_code":    code,
			"error_message": message,
		})
	if result.Error != nil {
		return fmt.Errorf("recommendation_logs SaveError id=%d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("recommendation_logs SaveError id=%d: %w", id, gorm.ErrRecordNotFound)
	}
	return nil
}

 
// ----------------------------------------------------------------
// CountRecentByUser
// ----------------------------------------------------------------
 
func (r *recommendationRepository) CountRecentByUser(ctx context.Context, userID uint64, since time.Time) (int64, error) {
	var count int64
	result := r.db.WithContext(ctx).
		Model(&model.RecommendationLog{}).
		Where("user_id = ? AND created_at >= ?", userID, since).
		Count(&count)
	if result.Error != nil {
		return 0, fmt.Errorf("recommendation_logs CountRecentByUser userID=%d: %w", userID, result.Error)
	}
	return count, nil
}