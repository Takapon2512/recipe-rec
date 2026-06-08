package model
 
import (
	"time"
 
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ----------------------------------------------------------------
// ステータス定数
// ----------------------------------------------------------------
 
// RecommendationStatus はジョブのライフサイクルを表す。
// pending → running → succeeded / failed の順に遷移する。
type RecommendationStatus string
 
const (
	RecommendationStatusPending   RecommendationStatus = "pending"
	RecommendationStatusRunning   RecommendationStatus = "running"
	RecommendationStatusSucceeded RecommendationStatus = "succeeded"
	RecommendationStatusFailed    RecommendationStatus = "failed"
)

// ----------------------------------------------------------------
// DBモデル
// ----------------------------------------------------------------
 
// RecommendationLog は recommendation_logs テーブルの1行に対応する。
// ジョブキューとログの両方を兼ねる設計で、status カラムで状態を管理する。
type RecommendationLog struct {
	ID uint64 `gorm:"primaryKey;autoIncrement"`
	UserID uint64               `gorm:"not null;index:idx_user_created,priority:1"`
	Status RecommendationStatus `gorm:"type:varchar(20);not null;default:'pending';index:idx_status_created,priority:1"`

	// リクエスト内容（フィルタ条件・件数）を JSON で保存する。
	// ワーカーがジョブを取得したあとにこのフィールドからプロンプトを組み立てる。
	RequestPayload datatypes.JSON `gorm:"type:json;not null"`

	// LLM の応答結果。pending / running 中は NULL。
	ResponsePayload datatypes.JSON `gorm:"type:json"`

	// 応答時間（ミリ秒）。succeeded 時のみセットされる。
	LatencyMs *int `gorm:"column:latency_ms"`

	// 失敗時のエラー情報。LLM_TIMEOUT / LLM_ERROR など。
	ErrorCode    *string `gorm:"type:varchar(50)"`
	ErrorMessage *string `gorm:"type:text"`
	
	CreatedAt time.Time      `gorm:"not null;index:idx_user_created,priority:2;index:idx_status_created,priority:2"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// ----------------------------------------------------------------
// リクエスト / レスポンス 用構造体
// ----------------------------------------------------------------
type RecommendationFilters struct {
	// 上限調理時間（分）。nil の場合は制限なし。
	MaxCookingTimeMin *int `json:"max_cooking_time_min,omitempty"`
 
	// ジャンル。japanese / western / chinese / other のいずれか。
	// nil の場合はジャンル指定なし（おまかせ）。
	Genre *string `json:"genre,omitempty"`
 
	// true の場合、期限切れ間近の食材を優先してプロンプトに渡す。
	PreferExpiring bool `json:"prefer_expiring"`
}

// CreateRecommendationRequest は POST /api/recommendations のリクエストボディ。
type CreateRecommendationRequest struct {
	Filters RecommendationFilters `json:"filters"`
 
	// 提案件数（1〜5）。0 はデフォルト値（3件）として扱う。
	Count int `json:"count"`
}

// CreateRecommendationResponse は POST /api/recommendations の 202 レスポンス。
// data フィールドにラップして返す。
type CreateRecommendationResponse struct {
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// ----------------------------------------------------------------
// LLM 応答の内部表現（response_payload の中身）
// ----------------------------------------------------------------
 
// RecommendationResultPayload は response_payload カラムに保存する JSON の構造。
// succeeded 時に RecommendationLog.ResponsePayload からデシリアライズして使う。
type RecommendationResultPayload struct {
	Recipes []RecommendedRecipe `json:"recipes"`
}

// RecommendedRecipe は LLM が提案した1レシピ分のデータ。
type RecommendedRecipe struct {
	Title          string                `json:"title"`
	CookingTimeMin int                   `json:"cooking_time_min"`
	Genre          string                `json:"genre"`
	Ingredients    []RecipeIngredientLLM `json:"ingredients"`
	// 在庫にない材料。買い物リスト候補として画面に表示する。
	MissingIngredients []MissingIngredient `json:"missing_ingredients"`
	Steps              []RecipeStepLLM     `json:"steps"`
}

// RecipeIngredientLLM は LLM 提案レシピの材料1件。
// from_inventory が true の場合、現在の在庫から使用できる材料を示す。
type RecipeIngredientLLM struct {
	Name          string  `json:"name"`
	Quantity      float64 `json:"quantity"`
	Unit          string  `json:"unit"`
	FromInventory bool    `json:"from_inventory"`
}

// MissingIngredient は在庫にない材料（買い物リスト候補）。
type MissingIngredient struct {
	Name     string  `json:"name"`
	Quantity float64 `json:"quantity"`
	Unit     string  `json:"unit"`
}

// RecipeStepLLM は LLM 提案レシピの調理手順1ステップ。
type RecipeStepLLM struct {
	StepNo      int    `json:"step_no"`
	Description string `json:"description"`
	DurationMin int    `json:"duration_min,omitempty"`
}

// GetJobResponse は GET /api/recommendations/:job_id のレスポンス。
type GetJobResponse struct {
	JobID     string                       `json:"job_id"`
	Status    string                       `json:"status"`
	CreatedAt time.Time                    `json:"created_at"`
	UpdatedAt time.Time                    `json:"updated_at"`
	Result    *RecommendationResultPayload `json:"result,omitempty"`
	Error     *JobError                    `json:"error,omitempty"`
}

// JobError は failed ジョブのエラー情報。
type JobError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}