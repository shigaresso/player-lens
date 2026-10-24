package adapter

import (
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

const (
	defaultReviews = 20
	maxReviews     = 100
	defaultPages   = 3
	maxPages       = 5
)

// normalizeToolInput は MCP 用の既定値と上限を適用し、収集操作へ渡す条件を作る。
// 入力を変更せず、不正時はゼロ値と ErrInvalidArgument を返す。
func normalizeToolInput(input toolInput) (steamdomain.AppID, steamdomain.CollectionOptions, error) {
	if input.AppID == 0 || input.MaxReviews < 0 || input.MaxReviews > maxReviews || input.MaxPages < 0 || input.MaxPages > maxPages {
		return 0, steamdomain.CollectionOptions{}, steamdomain.ErrInvalidArgument
	}
	if input.MaxReviews == 0 {
		input.MaxReviews = defaultReviews
	}
	if input.MaxPages == 0 {
		input.MaxPages = defaultPages
	}
	reviewType, err := steamdomain.ParseReviewType(input.ReviewType)
	if err != nil {
		return 0, steamdomain.CollectionOptions{}, steamdomain.ErrInvalidArgument
	}
	options, err := steamdomain.NormalizeCollectionOptions(steamdomain.CollectionOptions{
		ReviewType: reviewType, Language: input.Language, PageSize: input.PageSize, MaxReviews: input.MaxReviews, MaxPages: input.MaxPages,
	})
	if err != nil {
		return 0, steamdomain.CollectionOptions{}, steamdomain.ErrInvalidArgument
	}
	return input.AppID, options, nil
}
