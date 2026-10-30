// Package protocol は MCP の入力・応答と公開 schema の表現を担当する。
package protocol

import (
	"encoding/json"
	"fmt"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

const (
	defaultReviews = 20
	maxReviews     = 100
	defaultPages   = 3
	maxPages       = 5
)

// ParseInput は MCP 入力を検査・正規化して有効な収集条件を返す。
// 不正時はゼロ値と steamdomain.ErrInvalidArgument を保持したエラーを返す。
func ParseInput(data []byte) (steamdomain.AppID, steamdomain.CollectionOptions, error) {
	invalid := func() (steamdomain.AppID, steamdomain.CollectionOptions, error) {
		return 0, steamdomain.CollectionOptions{}, fmt.Errorf("invalid MCP collection input: %w", steamdomain.ErrInvalidArgument)
	}
	var input toolInput
	if err := json.Unmarshal(data, &input); err != nil {
		return invalid()
	}

	appID, options, err := normalizeToolInput(input)
	if err != nil {
		return invalid()
	}

	return appID, options, nil
}

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

	options, err := steamdomain.NewCollectionOptions(steamdomain.CollectionOptionsConfig{
		ReviewType: reviewType, Language: input.Language, PageSize: input.PageSize, MaxReviews: input.MaxReviews, MaxPages: input.MaxPages,
	})
	if err != nil {
		return 0, steamdomain.CollectionOptions{}, steamdomain.ErrInvalidArgument
	}

	return input.AppID, options, nil
}
