package repository

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"gorm.io/gorm"
)

type InventoryRepository struct {
	db *gorm.DB
}

// suggestRow は Suggest クエリの中間結果。Category 名の解決前に使う内部型。
type suggestRow struct {
	Name                    string    `gorm:"column:name"`
	FrequentCategoryID      *int      `gorm:"column:frequent_category_id"`
	FrequentStorageLocation *string   `gorm:"column:frequent_storage_location"`
	LastUsedAt              time.Time `gorm:"column:last_used_at"`
}

func NewInventoryRepository(db *gorm.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

// List は在庫一覧をフィルタ・ソート・ページネーション付きで返す。
// 戻り値は (items, total件数, error)。
func (r *InventoryRepository) List(userID uint64, params model.ListInventoryParams) ([]model.InventoryItem, int64, error) {
	q := r.db.Model(&model.InventoryItem{}).
		Preload("Category").
		Where("user_id = ?", userID)

	// --- フィルタ ---
	if params.Q != "" {
		q = q.Where("name LIKE ?", "%"+params.Q+"%")
	}

	if params.CategoryID != nil {
		if *params.CategoryID == 0 {
			// category_id=0 は未分類（NULL）として扱う
			q = q.Where("category_id IS NULL")
		} else {
			q = q.Where("category_id = ?", *params.CategoryID)
		}
	}

	if params.StorageLocation != "" {
		q = q.Where("storage_location = ?", params.StorageLocation)
	}

	// --- 総件数取得 ---
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("inventory count 失敗: %w", err)
	}

	// --- ソート ---
	q = applySort(q, params.Sort)

	// --- ページネーション ---
	offset := (params.Page - 1) * params.PerPage
	q = q.Limit(params.PerPage).Offset(offset)

	var items []model.InventoryItem
	if err := q.Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("inventory list 取得失敗: %w", err)
	}

	return items, total, nil
}

// applySort は sort パラメータに応じた ORDER BY 句を付与する。
// 先頭 "-" で降順。expires_at 昇順のとき NULL を末尾にまとめる。
func applySort(q *gorm.DB, sort string) *gorm.DB {
	desc := strings.HasPrefix(sort, "-")
	field := strings.TrimPrefix(sort, "-")

	dir := "ASC"
	if desc {
		dir = "DESC"
	}

	switch field {
	case "name":
		return q.Order(fmt.Sprintf("name %s", dir))
	case "expires_at":
		if desc {
			// 降順：NULL を先頭に
			return q.Order("expires_at IS NULL DESC, expires_at DESC")
		}
		// 昇順：NULL を末尾に（設計書 §4.1 ソート挙動の特記事項）
		return q.Order("expires_at IS NULL ASC, expires_at ASC")
	case "created_at":
		return q.Order(fmt.Sprintf("created_at %s", dir))
	default:
		// デフォルトは期限昇順（NULL末尾）
		return q.Order("expires_at IS NULL ASC, expires_at ASC")
	}
}

// Create は在庫レコードを1件挿入し、カテゴリを Preload した状態で返す。
func (r *InventoryRepository) Create(item *model.InventoryItem) error {
	if err := r.db.Create(item).Error; err != nil {
		return fmt.Errorf("inventory create 失敗: %w", err)
	}

	// Category を Preload して返す
	if item.CategoryID != nil {
		if err := r.db.Preload("Category").First(item, item.ID).Error; err != nil {
			return fmt.Errorf("inventory preload 失敗: %w", err)
		}
	}
	return nil
}

// FindByIDAndUserID は指定 id かつ user_id が一致するレコードを返す。
// 見つからない場合は ErrNotFound を返す。
func (r *InventoryRepository) FindByIDAndUserID(id, userID uint64) (*model.InventoryItem, error) {
	var item model.InventoryItem

	err := r.db.Preload("Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

// CategoryExists は category_id が categories テーブルに存在するか確認する。
func (r *InventoryRepository) CategoryExists(categoryID int) (bool, error) {
	var count int64
	err := r.db.Model(&model.Category{}).
		Where("id = ?", categoryID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("category 存在確認失敗: %w", err)
	}
	return count > 0, nil
}

// Update は渡された InventoryItem の内容でDBを更新する。
// service 層で非 nil フィールドのみ上書きした構造体を渡すこと。
// Select("*") + Omit("created_at", "deleted_at") を使うことで、
// category_id = NULL への更新（カテゴリ解除）も正しく反映される。
// GORM のゼロ値スキップ問題を回避するための明示的な全カラム指定。
func (r *InventoryRepository) Update(item *model.InventoryItem) error {
	result := r.db.Model(item).Select("*").Omit("created_at", "deleted_at").Save(item)

	if result.Error != nil {
		return fmt.Errorf("inventory update 失敗: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// SoftDelete は指定した id かつ user_id に紐づく在庫を論理削除する。
// deleted_at に現在時刻をセットする。
// 存在しない・既に削除済み・他ユーザーのリソースの場合は ErrNotFound を返す（設計書 §10 認可ルール）。
func (r *InventoryRepository) SoftDelete(userID, id uint64) error {
	result := r.db.Model(&model.InventoryItem{}).Where("id = ? AND user_id = ?", id, userID).Update("deleted_at", time.Now().UTC())

	if result.Error != nil {
		return fmt.Errorf("inventory delete 失敗: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// ListExpiring は期限切れ間近の在庫一覧を返す。
// withinDays 日以内に expires_at が到来するレコードを期限昇順で返す。
// 論理削除済みは除外。
func (r *InventoryRepository) ListExpiring(userID uint64, withinDays int) ([]model.InventoryItem, error) {
	var items []model.InventoryItem

	err := r.db.Preload("Category").
		Where("user_id = ? AND expires_at IS NOT NULL AND expires_at <= DATE_ADD(CURDATE(), INTERVAL ? DAY)", userID, withinDays).
		Order("expires_at ASC").
		Find(&items).Error

	if err != nil {
		return nil, fmt.Errorf("inventory expiring 取得失敗: %w", err)
	}

	return items, nil
}

// Suggest は q に部分一致する商品名サジェストを返す。
// 過去30日以内の登録履歴（論理削除済み含む）を母集団とし、
// name でグループ化して最頻 category_id・storage_location と最終登録日時を集計する。
func (r *InventoryRepository) Suggest(userID uint64, q string, limit int) ([]model.SuggestItem, error) {
	var rows []suggestRow

	err := r.db.Unscoped().
		Model(&model.InventoryItem{}).
		Select(`
			name,
			MAX(created_at) AS last_used_at,
			(SELECT category_id FROM inventory_items i2
			WHERE i2.user_id = ? AND i2.name = inventory_items.name
			AND i2.created_at >= NOW() - INTERVAL 30 DAY
			GROUP BY category_id
			ORDER BY COUNT(*) DESC, MAX(created_at) DESC
			LIMIT 1) AS frequent_category_id,
			(SELECT storage_location FROM inventory_items i3
			WHERE i3.user_id = ? AND i3.name = inventory_items.name
			AND i3.created_at >= NOW() - INTERVAL 30 DAY
			GROUP BY storage_location
			ORDER BY COUNT(*) DESC, MAX(created_at) DESC
			LIMIT 1) AS frequent_storage_location`,
			userID, userID).
		Where("user_id = ? AND name LIKE ? AND created_at >= NOW() - INTERVAL 30 DAY",
			userID, "%"+q+"%").
		Group("name").
		Order("last_used_at DESC").
		Limit(limit).
		Scan(&rows).Error

	if err != nil {
		return nil, fmt.Errorf("inventory suggest 取得失敗: %w", err)
	}

	// frequent_category_id からカテゴリ名を解決する
	categoryIDs := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.FrequentCategoryID != nil {
			categoryIDs = append(categoryIDs, *row.FrequentCategoryID)
		}
	}

	categoryMap := make(map[int]model.Category)
	if len(categoryIDs) > 0 {
		var categories []model.Category
		if err := r.db.Where("id IN ?", categoryIDs).Find(&categories).Error; err != nil {
			return nil, fmt.Errorf("category 取得失敗: %w", err)
		}
		for _, c := range categories {
			categoryMap[c.ID] = c
		}
	}

	// SuggestItem に変換
	result := make([]model.SuggestItem, len(rows))
	for i, row := range rows {
		item := model.SuggestItem{
			Name:                    row.Name,
			FrequentStorageLocation: row.FrequentStorageLocation,
			LastUsedAt:              row.LastUsedAt.UTC().Format(time.RFC3339),
		}

		if row.FrequentCategoryID != nil {
			if c, ok := categoryMap[*row.FrequentCategoryID]; ok {
				item.FrequentCategory = &model.CategoryResponse{
					ID:   c.ID,
					Name: c.Name,
				}
			}
		}

		result[i] = item
	}

	return result, nil
}

// GetSummary は在庫件数サマリーを1クエリで集計して返す。
func (r *InventoryRepository) GetSummary(userID uint64, expiringWithinDays int) (*model.InventorySummary, error) {
	var row model.InventorySummary

	err := r.db.Model(&model.InventoryItem{}).
		Select(`
			COUNT(*) AS total_count,
			SUM(CASE WHEN expires_at >= CURDATE()
			         AND expires_at <= DATE_ADD(CURDATE(), INTERVAL ? DAY)
			         THEN 1 ELSE 0 END) AS expiring_count,
			SUM(CASE WHEN expires_at < CURDATE() THEN 1 ELSE 0 END) AS expired_count,
			SUM(CASE WHEN expires_at IS NULL THEN 1 ELSE 0 END)     AS no_expiry_count`,
			expiringWithinDays).
		Where("user_id = ?", userID).
		Scan(&row).Error
	if err != nil {
		return nil, fmt.Errorf("inventory summary 取得失敗: %w", err)
	}

	return &row, nil
}

// Restore は論理削除済みの在庫を復元する（deleted_at を NULL に戻す）。
// 復元後のレコードを Preload 付きで返す。
// - レコードが存在しない / 他ユーザーのリソース → ErrNotFound
// - 既に復元済み（deleted_at IS NULL） → ErrConflict
func (r *InventoryRepository) Restore(userID, id uint64) (*model.InventoryItem, error) {
	result := r.db.Unscoped().
		Model(&model.InventoryItem{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NOT NULL", id, userID).
		Update("deleted_at", nil)
	
	if result.Error != nil {
		return nil, fmt.Errorf("inventory restore 失敗: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		// 存在確認で 404 と 409 を振り分ける
		var count int64
		r.db.Unscoped().
			Model(&model.InventoryItem{}).
			Where("id = ? AND user_id = ?", id, userID).
			Count(&count)

		if count == 0 {
			return nil, ErrNotFound
		}
		return nil, ErrConflict
	}

	var item model.InventoryItem
	if err := r.db.Preload("Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&item).Error; err != nil {
		return nil, fmt.Errorf("inventory restore 後の取得失敗: %w", err)
	}

	return &item, nil
}
