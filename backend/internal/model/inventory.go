package model

import (
	"time"

	"gorm.io/gorm"
)

// InventoryItem は inventory_items テーブルのモデル。
type InventoryItem struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement"             json:"id"`
	UserID          uint64         `gorm:"not null;index"                       json:"-"`
	CategoryID      *int           `gorm:"default:null"                         json:"category_id"`
	Category        *Category      `gorm:"foreignKey:CategoryID"                json:"category"`
	Name            string         `gorm:"size:100;not null"                    json:"name"`
	Quantity        float64        `gorm:"type:decimal(10,2);not null"          json:"quantity"`
	Unit            string         `gorm:"size:20;not null"                     json:"unit"`
	PurchasedAt     *time.Time     `gorm:"type:date"                            json:"purchased_at"`
	ExpiresAt       *time.Time     `gorm:"type:date"                            json:"expires_at"`
	StorageLocation *string        `gorm:"size:20"                              json:"storage_location"`
	Memo            *string        `gorm:"type:text"                            json:"memo"`
	CreatedAt       time.Time      `                                            json:"created_at"`
	UpdatedAt       time.Time      `                                            json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index"                                json:"-"`
}

func (InventoryItem) TableName() string { return "inventory_items" }

// InventoryResponse はレスポンス用の構造体。
type InventoryResponse struct {
	ID              uint64            `json:"id"`
	Name            string            `json:"name"`
	Category        *CategoryResponse `json:"category"`
	Quantity        float64           `json:"quantity"`
	Unit            string            `json:"unit"`
	PurchasedAt     *string           `json:"purchased_at"`
	ExpiresAt       *string           `json:"expires_at"`
	StorageLocation *string           `json:"storage_location"`
	Memo            *string           `json:"memo"`
	CreatedAt       string            `json:"created_at"`
	UpdatedAt       string            `json:"updated_at"`
}

// CategoryResponse はネストして返すカテゴリ用。
type CategoryResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ValidUnits は許容する単位の一覧。
var ValidUnits = map[string]bool{
	"個": true, "g": true, "kg": true,
	"ml": true, "l": true, "本": true, "枚": true,
}

// ValidStorageLocations は許容する保管場所の一覧。
var ValidStorageLocations = map[string]bool{
	"fridge": true, "freezer": true, "pantry": true,
}

// CreateInventoryRequest は POST /api/inventory のリクエスト。
type CreateInventoryRequest struct {
	Name            string  `json:"name"             binding:"required,min=1,max=100"`
	CategoryID      *int    `json:"category_id"`
	Quantity        float64 `json:"quantity"         binding:"required,gt=0"`
	Unit            string  `json:"unit"             binding:"required"`
	PurchasedAt     *string `json:"purchased_at"`
	ExpiresAt       *string `json:"expires_at"`
	StorageLocation *string `json:"storage_location"`
	Memo            *string `json:"memo" binding:"omitempty,max=500"`
}

// ListInventoryParams は GET /api/inventory のクエリパラメータ。
type ListInventoryParams struct {
	Page            int    `form:"page"`
	PerPage         int    `form:"per_page"`
	CategoryID      *int   `form:"category_id"`
	StorageLocation string `form:"storage_location"`
	Q               string `form:"q"`
	Sort            string `form:"sort"`
}

// Pagination はページネーションメタデータ。
type Pagination struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// UpdateInventoryItemRequest は PATCH /api/inventory/{id} のリクエストボディ。
// 送信されたフィールドのみ更新する部分更新（omitempty は使わず *型 でゼロ値と未送信を区別する）。
// NOTE: binding タグは ShouldBindJSON では使われない。ハンドラが map[string]json.RawMessage で
// 手動パースするため、バリデーションは service 層で行っている。
type UpdateInventoryItemRequest struct {
	Name            *string  `json:"name"`
	CategoryID      *int     `json:"category_id"` // null を明示的に送るとカテゴリ解除
	ClearCategoryID bool     `json:"-"`           // "category_id": null が明示送信された場合に true をセット
	Quantity        *float64 `json:"quantity"`
	Unit            *string  `json:"unit"`
	PurchasedAt     *string  `json:"purchased_at"`
	ExpiresAt       *string  `json:"expires_at"`
	StorageLocation *string  `json:"storage_location"`
	Memo            *string  `json:"memo"`
}

// ExpiringItem は GET /api/inventory/expiring のレスポンス1件。
type ExpiringItem struct {
	ID            uint64            `json:"id"`
	Name          string            `json:"name"`
	ExpiresAt     string            `json:"expires_at"`     // "YYYY-MM-DD"
	DaysRemaining int               `json:"days_remaining"` // 負値=期限切れ
	Category      *CategoryResponse `json:"category"`
}

// ExpiringParams は GET /api/inventory/expiring のクエリパラメータ。
type ExpiringParams struct {
	WithinDays *int `form:"within_days" binding:"omitempty,min=1"` // 省略時はサービス層でデフォルト 3 を適用
}

// SuggestItem は GET /api/inventory/suggest のレスポンス1件。
type SuggestItem struct {
	Name                    string            `json:"name"`
	FrequentCategory        *CategoryResponse `json:"frequent_category"`
	FrequentStorageLocation *string           `json:"frequent_storage_location"`
	LastUsedAt              string            `json:"last_used_at"` // ISO 8601 UTC
}

// SuggestParams は GET /api/inventory/suggest のクエリパラメータ。
type SuggestParams struct {
	Q     string `form:"q"     binding:"required,min=1"`
	Limit *int   `form:"limit" binding:"omitempty,min=1,max=12"` // 省略時はサービス層でデフォルト 5 を適用
}

// InventorySummary は GET /api/inventory/summary のレスポンス。
type InventorySummary struct {
	TotalCount    int64 `json:"total_count"`
	ExpiringCount int64 `json:"expiring_count"`  // 当日〜3日以内に期限が来るもの
	ExpiredCount  int64 `json:"expired_count"`   // 既に期限切れ（expires_at < 今日）
	NoExpiryCount int64 `json:"no_expiry_count"` // expires_at IS NULL
}
