package service

import (
	"errors"
	"fmt"

	"github.com/Takapon2512/recipe-recommend/backend/internal/cognito"
	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"github.com/Takapon2512/recipe-recommend/backend/internal/repository"
	"github.com/go-sql-driver/mysql"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

const mysqlDuplicateEntryCode = 1062

// GetOrCreate はJITプロビジョニングの中核。
// cognito_sub でユーザーを検索し、未登録であれば新規作成して返す。
func (s *UserService) GetOrCreate(claims *cognito.Claims) (*model.User, error) {
	user, err := s.repo.FindByCognitoSub(claims.Sub)
	if err == nil {
		// 既存ユーザー
		return user, nil
	}

	if !repository.IsNotFound(err) {
		return nil, fmt.Errorf("ユーザー検索失敗: %w", err)
	}

	// 新規ユーザー作成
	newUser := &model.User{
		CognitoSub: claims.Sub,
		Email:      claims.Email,
		Provider:   claims.Provider,
	}

	if err := s.repo.Create(newUser); err != nil {
		// 一意制約違反 → 別goroutineが先にInsertした → 再取得して返す
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntryCode {
			return s.repo.FindByCognitoSub(claims.Sub)
		}

		return nil, fmt.Errorf("ユーザー作成失敗: %w", err)
	}

	return newUser, nil
}
