package repository

import (
	"errors"

	"github.com/Takapon2512/recipe-recommend/backend/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// FindByCognitoSub は cognito_sub でユーザーを検索する。未登録の場合は gorm.ErrRecordNotFound を返す。
func (r *UserRepository) FindByCognitoSub(cognitoSub string) (*model.User, error) {
	var user model.User

	err := r.db.Where("cognito_sub = ?", cognitoSub).First(&user).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Create はユーザーを新規登録する。
func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// IsNotFound は gorm.ErrRecordNotFound かどうかを判定する。
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
