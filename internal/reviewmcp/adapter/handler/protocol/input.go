// Package protocol は MCP の入力・応答と公開 schema の表現を担当する。
package protocol

import (
	"encoding/json"
	"errors"
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
// 不正時はゼロ値と steamdomain.ErrInvalidArgument・解析や条件検査の原因を保持したエラーを返す。
// エラーは内部診断用であり、クライアントへそのまま公開しない。
func ParseInput(data []byte) (steamdomain.AppID, steamdomain.CollectionOptions, error) {
	var input toolInput
	if err := json.Unmarshal(data, &input); err != nil {
		return 0, steamdomain.CollectionOptions{}, fmt.Errorf("decode MCP collection input: %w", errors.Join(steamdomain.ErrInvalidArgument, err))
	}

	appID, options, err := normalizeToolInput(input)
	if err != nil {
		return 0, steamdomain.CollectionOptions{}, fmt.Errorf("prepare MCP collection conditions: %w", err)
	}

	return appID, options, nil
}

// normalizeToolInput は MCP 用の既定値と上限を適用し、収集操作へ渡す条件を作る。
// 入力を変更せず、不正時はゼロ値と ErrInvalidArgument を返す。
func normalizeToolInput(input toolInput) (steamdomain.AppID, steamdomain.CollectionOptions, error) {
	if input.AppID == 0 {
		return 0, steamdomain.CollectionOptions{}, fmt.Errorf("app_id must be positive: %w", steamdomain.ErrInvalidArgument)
	}

	if input.MaxReviews < 0 || input.MaxReviews > maxReviews {
		return 0, steamdomain.CollectionOptions{}, fmt.Errorf("max_reviews must be between 0 and %d: %w", maxReviews, steamdomain.ErrInvalidArgument)
	}

	if input.MaxPages < 0 || input.MaxPages > maxPages {
		return 0, steamdomain.CollectionOptions{}, fmt.Errorf("max_pages must be between 0 and %d: %w", maxPages, steamdomain.ErrInvalidArgument)
	}

	if input.MaxReviews == 0 {
		input.MaxReviews = defaultReviews
	}

	if input.MaxPages == 0 {
		input.MaxPages = defaultPages
	}

	reviewType, err := steamdomain.ParseReviewType(input.ReviewType)
	if err != nil {
		return 0, steamdomain.CollectionOptions{}, err
	}

	options, err := steamdomain.NewCollectionOptions(steamdomain.CollectionOptionsConfig{
		ReviewType: reviewType, Language: input.Language, PageSize: input.PageSize, MaxReviews: input.MaxReviews, MaxPages: input.MaxPages,
	})
	if err != nil {
		return 0, steamdomain.CollectionOptions{}, err
	}

	return input.AppID, options, nil
}
