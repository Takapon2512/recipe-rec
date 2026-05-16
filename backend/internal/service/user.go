package service

import (
	"fmt"

	"github.com/Takapon2512/recipe-recommend/backend/internal/cognito"
	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"github.com/Takapon2512/recipe-recommend/backend/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

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
		return nil, fmt.Errorf("ユーザー作成失敗: %w", err)
	}

	return newUser, nil
}
