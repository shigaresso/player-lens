package domain

import "fmt"

// MaxCollectionReviews は1回の収集で返すレビュー数のアプリケーション側の上限を表す。
const MaxCollectionReviews = 10000

// MaxCollectionPages は1回の収集で取得するページ数のアプリケーション側の上限を表す。
const MaxCollectionPages = 100

// CollectionOptionsConfig は未検証の収集条件と表示用の設定コピーを表す。
// ゼロ値の入力は全評価・英語・100件/ページ・1000件・100ページの指定となる。
//
//contractcheck:data 生成前の入力と表示用の設定コピーを表すデータ
type CollectionOptionsConfig struct {
	ReviewType ReviewType
	Language   string
	PageSize   int
	MaxReviews int
	MaxPages   int
}

// normalizeCollectionOptions は通信せずに収集条件の既定値を適用し、上限を検査する。
// 入力を変更せず、不正な場合はゼロ値と ErrInvalidArgument を返す。
// 評価種別・言語・ページ件数には NormalizePageOptions の規則を使う。
func normalizeCollectionOptions(options CollectionOptionsConfig) (CollectionOptionsConfig, error) {
	defaults := defaultCollectionOptionsConfig()
	if options.PageSize == 0 {
		options.PageSize = defaults.PageSize
	}

	page, err := NormalizePageOptions(PageOptions{ReviewType: options.ReviewType, Language: options.Language, PageSize: options.PageSize})
	if err != nil {
		return CollectionOptionsConfig{}, fmt.Errorf("prepare collection page options: %w", err)
	}

	options.Language = page.Language
	options.ReviewType = page.ReviewType
	if options.MaxReviews == 0 {
		options.MaxReviews = defaults.MaxReviews
	}

	if options.MaxPages == 0 {
		options.MaxPages = defaults.MaxPages
	}

	if options.MaxReviews < 1 || options.MaxReviews > MaxCollectionReviews {
		return CollectionOptionsConfig{}, fmt.Errorf("collection review limit must be between 1 and %d: %w", MaxCollectionReviews, ErrInvalidArgument)
	}

	if options.MaxPages < 1 || options.MaxPages > MaxCollectionPages {
		return CollectionOptionsConfig{}, fmt.Errorf("collection page limit must be between 1 and %d: %w", MaxCollectionPages, ErrInvalidArgument)
	}

	return options, nil
}

// CollectionOptions は生成時に検査・正規化した有効な収集条件を保持する。
// ゼロ値も全評価・英語・100件/ページ・1000件・100ページの有効な条件を表す。
// 設定の取得には Config を使い、変更する場合は新しい値を生成する。
//
//contractcheck:controlled
type CollectionOptions struct {
	config CollectionOptionsConfig
}

// NewCollectionOptions は設定を検査・正規化してコピーを保持する。
// 入力を変更せず、不正ならゼロ値と ErrInvalidArgument を保持するエラーを返す。
// ゼロ値自体も有効な既定条件のため、呼び出し側は必ず戻りエラーを処理する。
func NewCollectionOptions(config CollectionOptionsConfig) (CollectionOptions, error) {
	normalized, err := normalizeCollectionOptions(config)
	if err != nil {
		return CollectionOptions{}, err
	}

	if normalized == defaultCollectionOptionsConfig() {
		return CollectionOptions{}, nil
	}

	return CollectionOptions{config: normalized}, nil
}

// Config は設定のコピーを返す。コピーの変更は元の値に影響しない。
// ゼロ値からも既定条件を返し、検査・正規化・通信を繰り返さない。
func (options CollectionOptions) Config() CollectionOptionsConfig {
	if options.config == (CollectionOptionsConfig{}) {
		return defaultCollectionOptionsConfig()
	}

	return options.config
}

func defaultCollectionOptionsConfig() CollectionOptionsConfig {
	return CollectionOptionsConfig{ReviewType: ReviewTypeAll, Language: "english", PageSize: MaxPageSize, MaxReviews: 1000, MaxPages: MaxCollectionPages}
}

// DefaultCollectionOptions は有効な既定の収集条件を生成する。
// 内部のゼロ表現を返し、Config は全評価・英語・100件/ページ・1000件・100ページを返す。
func DefaultCollectionOptions() CollectionOptions {
	return CollectionOptions{}
}
