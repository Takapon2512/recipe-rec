package repository

import "errors"

// ErrNotFound は対象レコードが存在しない・論理削除済み・他ユーザーのリソースの場合のエラー。
var ErrNotFound = errors.New("not found")
