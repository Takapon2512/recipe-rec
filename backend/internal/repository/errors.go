package repository

import "errors"

// ErrNotFound は対象レコードが存在しない・論理削除済み・他ユーザーのリソースの場合のエラー。
var ErrNotFound = errors.New("not found")

// ErrConflict はすでに更新済みである場合のエラー
var ErrConflict = errors.New("conflict")
