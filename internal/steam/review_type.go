package steam

import "fmt"

// EUserReviewsReviewType は Steam の全評価・おすすめ・非おすすめを表す int32 の列挙型。
// ゼロ値は全評価であり、未知の数値は取得条件と出力境界で拒否する。
type EUserReviewsReviewType int32

const (
	k_EUserReviewsReviewType_All      EUserReviewsReviewType = 0
	k_EUserReviewsReviewType_Positive EUserReviewsReviewType = 1
	k_EUserReviewsReviewType_Negative EUserReviewsReviewType = 2
)

const (
	// ReviewTypeAll は公式の All（0）を表す。
	ReviewTypeAll = k_EUserReviewsReviewType_All
	// ReviewTypePositive は公式の Positive（1）を表す。
	ReviewTypePositive = k_EUserReviewsReviewType_Positive
	// ReviewTypeNegative は公式の Negative（2）を表す。
	ReviewTypeNegative = k_EUserReviewsReviewType_Negative
)

// ParseReviewType は CLI・MCP の評価種別の文字列を Steam の数値列挙値に変換する。
// 空は全評価とし、未知値ではゼロ値と ErrInvalidArgument を返す。
func ParseReviewType(value string) (EUserReviewsReviewType, error) {
	switch value {
	case "", "all":
		return k_EUserReviewsReviewType_All, nil
	case "positive":
		return k_EUserReviewsReviewType_Positive, nil
	case "negative":
		return k_EUserReviewsReviewType_Negative, nil
	default:
		return 0, fmt.Errorf("parse Steam review type: %w", ErrInvalidArgument)
	}
}

// FormatReviewType は Steam の評価種別を CLI・MCP・JSON 用の文字列に変換する。
// 未知の数値では空文字と ErrInvalidArgument を返す。
func FormatReviewType(value EUserReviewsReviewType) (string, error) {
	switch value {
	case k_EUserReviewsReviewType_All:
		return "all", nil
	case k_EUserReviewsReviewType_Positive:
		return "positive", nil
	case k_EUserReviewsReviewType_Negative:
		return "negative", nil
	default:
		return "", fmt.Errorf("format Steam review type %d: %w", value, ErrInvalidArgument)
	}
}
