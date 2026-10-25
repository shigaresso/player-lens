package domain

import (
	"fmt"
	"regexp"
)

// MaxPageSize は Steam GetAppReviews が1ページで返すレビュー件数の上限を表す。
const MaxPageSize = 100

// FirstPageCursor は Steam GetAppReviews で先頭ページを指定するカーソルを表す。
const FirstPageCursor = "*"

const defaultPageSize = 20
const maxCursorBytes = 4096

var languagePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// PageOptions は1ページの取得条件を表す。ゼロ値は全評価・英語・先頭・20件となる。
type PageOptions struct {
	ReviewType ReviewType
	Language   string
	Cursor     string
	PageSize   int
}

// NormalizePageOptions は通信せずに既定値を補完し、取得条件を検査する。
// 件数は1〜MaxPageSize、言語は32文字以内の API コード構文、カーソルは4096バイト以内とする。
// 入力を変更せず、不正な場合はゼロ値と ErrInvalidArgument を返す。
// 戻り値の書き換え後の有効性や Steam 側の対応言語は保証しない。
func NormalizePageOptions(options PageOptions) (PageOptions, error) {
	switch options.ReviewType {
	case ReviewTypeAll, ReviewTypePositive, ReviewTypeNegative:
	default:
		return PageOptions{}, fmt.Errorf("invalid Steam review type: %w", ErrInvalidArgument)
	}

	if options.Language == "" {
		options.Language = "english"
	}

	if options.Cursor == "" {
		options.Cursor = FirstPageCursor
	}

	if options.PageSize == 0 {
		options.PageSize = defaultPageSize
	}

	if options.PageSize < 1 || options.PageSize > MaxPageSize {
		return PageOptions{}, fmt.Errorf("steam review page size must be between 1 and %d: %w", MaxPageSize, ErrInvalidArgument)
	}

	if !languagePattern.MatchString(options.Language) {
		return PageOptions{}, fmt.Errorf("invalid Steam review language code: %w", ErrInvalidArgument)
	}

	if len(options.Cursor) > maxCursorBytes {
		return PageOptions{}, fmt.Errorf("steam review cursor exceeds %d bytes: %w", maxCursorBytes, ErrInvalidArgument)
	}

	return options, nil
}
