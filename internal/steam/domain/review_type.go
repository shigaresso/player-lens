package domain

import "fmt"

// ReviewType は内部の取得条件に使う全評価・おすすめ・非おすすめの数値列挙型。
// 値は Steam の EUserReviewsReviewType に対応し、ゼロ値は全評価を表す。
// 未知の数値は取得条件と出力境界で拒否する。
//
//contractcheck:data 公開された数値の入力表現であり未知値は解析・正規化・出力境界で拒否する
type ReviewType EUserReviewsReviewType

const (
	// ReviewTypeAll は公式の All（0）を表す。
	ReviewTypeAll ReviewType = ReviewType(k_EUserReviewsReviewType_All)
	// ReviewTypePositive は公式の Positive（1）を表す。
	ReviewTypePositive ReviewType = ReviewType(k_EUserReviewsReviewType_Positive)
	// ReviewTypeNegative は公式の Negative（2）を表す。
	ReviewTypeNegative ReviewType = ReviewType(k_EUserReviewsReviewType_Negative)
)

// ParseReviewType は CLI・MCP の評価種別の文字列を内部の数値列挙値に変換する。
// 空は全評価とし、未知値ではゼロ値と ErrInvalidArgument を返す。
func ParseReviewType(value string) (ReviewType, error) {
	switch value {
	case "", "all":
		return ReviewTypeAll, nil
	case "positive":
		return ReviewTypePositive, nil
	case "negative":
		return ReviewTypeNegative, nil
	default:
		return 0, fmt.Errorf("parse Steam review type: %w", ErrInvalidArgument)
	}
}

// FormatReviewType は内部の評価種別を CLI・MCP・JSON 用の文字列に変換する。
// 未知の数値では空文字と ErrInvalidArgument を返す。
func FormatReviewType(value ReviewType) (string, error) {
	switch value {
	case ReviewTypeAll:
		return "all", nil
	case ReviewTypePositive:
		return "positive", nil
	case ReviewTypeNegative:
		return "negative", nil
	default:
		return "", fmt.Errorf("format Steam review type %d: %w", value, ErrInvalidArgument)
	}
}
