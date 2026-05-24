package service

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"github.com/Takapon2512/recipe-recommend/backend/internal/repository"
)

// ErrCategoryNotFound は指定 category_id が存在しない場合のエラー。
var ErrCategoryNotFound = errors.New("category not found")

// ErrNotFound はリソースが存在しない・削除済み・他ユーザー所有の場合のエラー。
var ErrNotFound = errors.New("not found")

// ExpiringWithinDays はサマリーの「期限間近」判定日数。
const ExpiringWithinDays = 3

// ValidationError はリクエスト値起因のバリデーションエラー。
// ハンドラ側で errors.As により 400 と 500 を区別する。
type ValidationError struct{ msg string }

func (e *ValidationError) Error() string { return e.msg }

func NewValidationError(msg string) error { return &ValidationError{msg: msg} }

type InventoryService struct {
	repo *repository.InventoryRepository
}

func NewInventoryService(repo *repository.InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

// ListResult は一覧取得結果。
type ListResult struct {
	Items      []model.InventoryItem
	Pagination model.Pagination
}

// List は在庫一覧を取得してページネーションメタを計算して返す。
func (s *InventoryService) List(userID uint64, params model.ListInventoryParams) (*ListResult, error) {
	if params.Page < 1 {
		params.Page = 1
	}

	perPage := params.PerPage
	if perPage < 1 {
		perPage = 20
	}

	if perPage > 100 {
		perPage = 100
	}
	params.PerPage = perPage

	items, total, err := s.repo.List(userID, params)
	if err != nil {
		return nil, err
	}

	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	if totalPages == 0 {
		totalPages = 1
	}

	return &ListResult{
		Items: items,
		Pagination: model.Pagination{
			Page:       params.Page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

// Create は在庫を新規登録する。
// category_id が指定されている場合は存在確認を行い、存在しない場合は ErrCategoryNotFound を返す。
func (s *InventoryService) Create(userID uint64, req model.CreateInventoryRequest) (*model.InventoryItem, error) {
	// unit バリデーション
	if !model.ValidUnits[req.Unit] {
		return nil, NewValidationError(fmt.Sprintf("invalid unit: %s", req.Unit))
	}

	// storage_location バリデーション
	if req.StorageLocation != nil {
		loc := strings.TrimSpace(*req.StorageLocation)
		if loc != "" && !model.ValidStorageLocations[loc] {
			return nil, NewValidationError(fmt.Sprintf("invalid storage_location: %s", loc))
		}
		if loc == "" {
			req.StorageLocation = nil
		} else {
			req.StorageLocation = &loc
		}
	}

	// name の trim
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, NewValidationError("name must not be blank")
	}

	// memo の trim & 空文字 → nil
	if req.Memo != nil {
		trimmed := strings.TrimSpace(*req.Memo)
		if trimmed == "" {
			req.Memo = nil
		} else {
			req.Memo = &trimmed
		}
	}

	// category_id 存在確認
	if req.CategoryID != nil {
		exists, err := s.repo.CategoryExists(*req.CategoryID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrCategoryNotFound
		}
	}

	// 日付パース
	purchasedAt, err := parseDate(req.PurchasedAt)
	if err != nil {
		return nil, NewValidationError(fmt.Sprintf("invalid purchased_at: %s", err))
	}
	expiresAt, err := parseDate(req.ExpiresAt)
	if err != nil {
		return nil, NewValidationError(fmt.Sprintf("invalid expires_at: %s", err))
	}

	item := &model.InventoryItem{
		UserID:          userID,
		CategoryID:      req.CategoryID,
		Name:            req.Name,
		Quantity:        req.Quantity,
		Unit:            req.Unit,
		PurchasedAt:     purchasedAt,
		ExpiresAt:       expiresAt,
		StorageLocation: req.StorageLocation,
		Memo:            req.Memo,
	}

	if err := s.repo.Create(item); err != nil {
		return nil, err
	}

	return item, nil
}

// GetInventoryItem は在庫を1件取得して返す。
// レスポンス整形（ToResponse）はハンドラー側で行う。
// 存在しない・論理削除済み・他ユーザーのリソースの場合は ErrNotFound を返す。
func (s *InventoryService) GetInventoryItem(userID, id uint64) (*model.InventoryItem, error) {
	item, err := s.repo.FindByIDAndUserID(id, userID)
	if err != nil {
		return nil, translateNotFound(err)
	}
	return item, nil
}

// UpdateInventoryItem は在庫を部分更新して更新後のレコードを返す。
// レスポンス整形（ToResponse）はハンドラー側で行う。
// req の非 nil フィールドのみ上書きする（ゼロ値との区別のためポインタ型を利用）。
//
// category_id の扱い:
//   - req.CategoryID != nil          → 存在確認のうえカテゴリ変更
//   - req.ClearCategoryID == true    → nil に更新（未分類化）
//   - どちらでもない                  → 変更なし
func (s *InventoryService) UpdateInventoryItem(userID, id uint64, req *model.UpdateInventoryItemRequest) (*model.InventoryItem, error) {
	// 現在のレコードを取得（認可チェック兼用）
	item, err := s.repo.FindByIDAndUserID(id, userID)
	if err != nil {
		return nil, translateNotFound(err)
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, NewValidationError("name must not be blank")
		}
		item.Name = name
	}

	if req.CategoryID != nil {
		exists, err := s.repo.CategoryExists(*req.CategoryID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrCategoryNotFound
		}
		item.CategoryID = req.CategoryID
	}

	if req.ClearCategoryID {
		item.CategoryID = nil
	}

	if req.Quantity != nil {
		if *req.Quantity <= 0 {
			return nil, NewValidationError("quantity must be greater than 0")
		}
		item.Quantity = *req.Quantity
	}

	if req.Unit != nil {
		if !model.ValidUnits[*req.Unit] {
			return nil, NewValidationError(fmt.Sprintf("invalid unit: %s", *req.Unit))
		}
		item.Unit = *req.Unit
	}

	if req.PurchasedAt != nil {
		t, err := parseDate(req.PurchasedAt)
		if err != nil {
			return nil, NewValidationError(fmt.Sprintf("invalid purchased_at: %s", err))
		}
		item.PurchasedAt = t
	}

	if req.ExpiresAt != nil {
		t, err := parseDate(req.ExpiresAt)
		if err != nil {
			return nil, NewValidationError(fmt.Sprintf("invalid expires_at: %s", err))
		}
		item.ExpiresAt = t
	}

	if req.StorageLocation != nil {
		loc := strings.TrimSpace(*req.StorageLocation)
		if loc != "" && !model.ValidStorageLocations[loc] {
			return nil, NewValidationError(fmt.Sprintf("invalid storage_location: %s", loc))
		}
		if loc == "" {
			item.StorageLocation = nil
		} else {
			item.StorageLocation = &loc
		}
	}

	if req.Memo != nil {
		trimmed := strings.TrimSpace(*req.Memo)
		if trimmed == "" {
			item.Memo = nil
		} else {
			item.Memo = &trimmed
		}
	}

	if err := s.repo.Update(item); err != nil {
		return nil, translateNotFound(err)
	}

	// Category を再 Preload（category_id 変更後の最新状態を返すため）
	updated, err := s.repo.FindByIDAndUserID(id, userID)
	if err != nil {
		return nil, translateNotFound(err)
	}

	return updated, nil
}

// DeleteInventoryItem は在庫を論理削除する。
// 存在しない・論理削除済み・他ユーザーのリソースの場合は ErrNotFound を返す。
func (s *InventoryService) DeleteInventoryItem(userID, id uint64) error {
	return translateNotFound(s.repo.SoftDelete(userID, id))
}

// GetExpiring は期限切れ間近の在庫一覧を返す。
// withinDays 未指定（nil）の場合はデフォルト 3 日を適用する。
func (s *InventoryService) GetExpiring(userID uint64, params model.ExpiringParams) ([]model.ExpiringItem, error) {
	withinDays := 3

	if params.WithinDays != nil {
		withinDays = *params.WithinDays
	}

	items, err := s.repo.ListExpiring(userID, withinDays)
	if err != nil {
		return nil, err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	result := make([]model.ExpiringItem, len(items))
	for i, item := range items {
		if item.ExpiresAt == nil {
			continue // または適切なデフォルト値
		}

		daysRemaining := int(item.ExpiresAt.Sub(today).Hours() / 24)

		var category *model.CategoryResponse
		if item.Category != nil {
			category = &model.CategoryResponse{
				ID:   item.Category.ID,
				Name: item.Category.Name,
			}
		}

		result[i] = model.ExpiringItem{
			ID:            item.ID,
			Name:          item.Name,
			ExpiresAt:     item.ExpiresAt.Format("2006-01-02"),
			DaysRemaining: daysRemaining,
			Category:      category,
		}
	}

	return result, nil
}

// Suggest は q に部分一致する商品名サジェストを返す。
// limit 未指定（nil）の場合はデフォルト 5 を適用する。
func (s *InventoryService) Suggest(userID uint64, params model.SuggestParams) ([]model.SuggestItem, error) {
	limit := 5
	if params.Limit != nil {
		limit = *params.Limit
	}

	items, err := s.repo.Suggest(userID, params.Q, limit)
	if err != nil {
		return nil, err
	}

	return items, nil
}

// GetSummary は在庫件数サマリーを返す。
func (s *InventoryService) GetSummary(userID uint64) (*model.InventorySummary, error) {
	row, err := s.repo.GetSummary(userID, ExpiringWithinDays)
	if err != nil {
		return nil, err
	}

	return &model.InventorySummary{
		TotalCount:    row.TotalCount,
		ExpiringCount: row.ExpiringCount,
		ExpiredCount:  row.ExpiredCount,
		NoExpiryCount: row.NoExpiryCount,
	}, nil
}

// translateNotFound は repository.ErrNotFound を service.ErrNotFound に変換する。
// それ以外のエラーはそのまま返す。
func translateNotFound(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

// parseDate は "YYYY-MM-DD" 文字列を *time.Time に変換する。nil または空文字の場合は nil を返す。
func parseDate(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return nil, fmt.Errorf("date must be YYYY-MM-DD format, got: %s", trimmed)
	}
	return &t, nil
}
