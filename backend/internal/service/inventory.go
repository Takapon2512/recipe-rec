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
	// ページ・件数の正規化はリポジトリ側で行うが、ここでも初期化しておく
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
		return nil, fmt.Errorf("invalid unit: %s", req.Unit)
	}

	// storage_location バリデーション
	if req.StorageLocation != nil {
		loc := strings.TrimSpace(*req.StorageLocation)
		if loc != "" && !model.ValidStorageLocations[loc] {
			return nil, fmt.Errorf("invalid storage_location: %s", loc)
		}
		if loc == "" {
			req.StorageLocation = nil
		} else {
			req.StorageLocation = &loc
		}
	}

	// name の trim
	req.Name = strings.TrimSpace(req.Name)

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
		return nil, fmt.Errorf("invalid purchased_at: %w", err)
	}
	expiresAt, err := parseDate(req.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("invalid expires_at: %w", err)
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
