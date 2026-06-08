package anthropic

import (
	"context"

	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
)

// Client は Anthropic Claude API の呼び出しを抽象化するインターフェース。
// service 層はこのインターフェースに依存し、具体実装には依存しない。
type Client interface {
	// Recommend は在庫リストとフィルタ条件をもとにレシピ提案を要求する。
	// タイムアウトは呼び出し元の ctx で制御する（推奨: 60秒）。
	Recommend(ctx context.Context, req *RecommendRequest) (*model.RecommendationResultPayload, error)
}

// RecommendRequest は Claude API へ渡すリクエストの内部表現。
// プロンプト組み立ては client 実装に委譲する。
type RecommendRequest struct {
	// 提案に使う在庫リスト（プロンプトに埋め込む）
	InventoryItems []InventoryItem

	// フィルタ条件
	Filters model.RecommendationFilters
 
	// 提案件数（1〜5）
	Count int
}

// InventoryItem はプロンプト用の在庫1件の表現。
type InventoryItem struct {
	Name      string  // 商品名
	Quantity  float64 // 数量
	Unit      string  // 単位
	ExpiresAt *string // 賞味期限（YYYY-MM-DD）。nil の場合は期限なし
}