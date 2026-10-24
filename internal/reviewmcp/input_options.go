package reviewmcp

import "github.com/shigaresso/player-lens/internal/steam"

const (
	defaultReviews = 20
	maxReviews     = 100
	defaultPages   = 3
	maxPages       = 5
)

// normalizeToolInput は MCP 用の既定値と上限を適用し、収集操作へ渡す条件を作る。
// 入力を変更せず、不正時はゼロ値と ErrInvalidArgument を返す。
func normalizeToolInput(input toolInput) (steam.AppID, steam.CollectionOptions, error) {
	if input.AppID == 0 || input.MaxReviews < 0 || input.MaxReviews > maxReviews || input.MaxPages < 0 || input.MaxPages > maxPages {
		return 0, steam.CollectionOptions{}, steam.ErrInvalidArgument
	}
	if input.MaxReviews == 0 {
		input.MaxReviews = defaultReviews
	}
	if input.MaxPages == 0 {
		input.MaxPages = defaultPages
	}
	reviewType, err := steam.ParseReviewType(input.ReviewType)
	if err != nil {
		return 0, steam.CollectionOptions{}, steam.ErrInvalidArgument
	}
	options, err := steam.NormalizeCollectionOptions(steam.CollectionOptions{
		ReviewType: reviewType, Language: input.Language, PageSize: input.PageSize, MaxReviews: input.MaxReviews, MaxPages: input.MaxPages,
	})
	if err != nil {
		return 0, steam.CollectionOptions{}, steam.ErrInvalidArgument
	}
	return input.AppID, options, nil
}
