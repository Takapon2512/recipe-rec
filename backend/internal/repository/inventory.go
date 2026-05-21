package repository

import (
	"fmt"
	"strings"

	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"gorm.io/gorm"
)

type InventoryRepository struct {
	db *gorm.DB
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
// 見つからない場合は gorm.ErrRecordNotFound。
func (r *InventoryRepository) FindByIDAndUserID(id, userID uint64) (*model.InventoryItem, error) {
	var item model.InventoryItem

	err := r.db.Preload("Category").
		Where("id = ? AND user_id = ?", id, userID).
		First(&item).Error
	if err != nil {
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
