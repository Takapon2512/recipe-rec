package handler

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

// friendlyMessage は validator.FieldError をユーザー向けの日本語メッセージに変換する。
func friendlyMessage(fe validator.FieldError) string {
	field := fieldLabel(fe.Field())

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s は必須です", field)
	case "gt":
		return fmt.Sprintf("%s は %s より大きい値を入力してください", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s は %s 以上の値を入力してください", field, fe.Param())
	case "lt":
		return fmt.Sprintf("%s は %s より小さい値を入力してください", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s は %s 以下の値を入力してください", field, fe.Param())
	case "min":
		return fmt.Sprintf("%s は %s 文字以上で入力してください", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s は %s 文字以内で入力してください", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s は次のいずれかを指定してください: %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s の値が不正です", field)
	}
}

// fieldLabel は構造体フィールド名を日本語ラベルに変換する。
func fieldLabel(field string) string {
	labels := map[string]string{
		"Name":            "商品名",
		"Quantity":        "数量",
		"Unit":            "単位",
		"CategoryID":      "カテゴリ",
		"PurchasedAt":     "購入日",
		"ExpiresAt":       "期限日",
		"StorageLocation": "保管場所",
		"Memo":            "メモ",
	}

	if label, ok := labels[field]; ok {
		return label
	}
	return field
}

// bindingErrorMessage は ShouldBindJSON / ShouldBindQuery のエラーを
// ユーザー向けメッセージ文字列に変換する。
// validator.ValidationErrors の場合は最初のエラーを日本語化して返す。
// それ以外（型不一致・JSON構文エラー等）は汎用メッセージを返す。
func bindingErrorMessage(err error) string {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		return friendlyMessage(ve[0])
	}
	return "リクエストの形式が正しくありません"
}
