package steam

import "fmt"

// MaxCollectionReviews は1回の収集で返すレビュー数のアプリケーション側の上限を表す。
const MaxCollectionReviews = 10000

// MaxCollectionPages は1回の収集で取得するページ数のアプリケーション側の上限を表す。
const MaxCollectionPages = 100

// CollectionOptions は収集条件を表す。ゼロ値は英語・100件/ページ・1000件・100ページとなる。
type CollectionOptions struct {
	Language   string
	PageSize   int
	MaxReviews int
	MaxPages   int
}

// NormalizeCollectionOptions は通信せずに収集条件の既定値を適用し、上限を検査する。
// 入力を変更せず、不正な場合はゼロ値と ErrInvalidArgument を返す。
// 言語とページ件数には NormalizePageOptions の規則を使う。
func NormalizeCollectionOptions(options CollectionOptions) (CollectionOptions, error) {
	if options.PageSize == 0 {
		options.PageSize = MaxPageSize
	}
	page, err := NormalizePageOptions(PageOptions{Language: options.Language, PageSize: options.PageSize})
	if err != nil {
		return CollectionOptions{}, fmt.Errorf("prepare collection page options: %w", err)
	}
	options.Language = page.Language
	if options.MaxReviews == 0 {
		options.MaxReviews = 1000
	}
	if options.MaxPages == 0 {
		options.MaxPages = MaxCollectionPages
	}
	if options.MaxReviews < 1 || options.MaxReviews > MaxCollectionReviews {
		return CollectionOptions{}, fmt.Errorf("collection review limit must be between 1 and %d: %w", MaxCollectionReviews, ErrInvalidArgument)
	}
	if options.MaxPages < 1 || options.MaxPages > MaxCollectionPages {
		return CollectionOptions{}, fmt.Errorf("collection page limit must be between 1 and %d: %w", MaxCollectionPages, ErrInvalidArgument)
	}
	return options, nil
}
