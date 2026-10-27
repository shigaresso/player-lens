package adapter

import (
	"encoding/json"
	"errors"
	"fmt"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type reviewPageResponse struct {
	Response *struct {
		Reviews []reviewPageReview `json:"reviews"`
		Cursor  string             `json:"cursor"`
	} `json:"response"`
}

type reviewPageReview struct {
	RecommendationID string                       `json:"recommendationid"`
	Text             string                       `json:"review"`
	Language         string                       `json:"language"`
	Recommended      bool                         `json:"voted_up"`
	TimestampCreated steamdomain.TimestampCreated `json:"timestamp_created"`
	TimestampUpdated steamdomain.TimestampUpdated `json:"timestamp_updated"`
}

// decodeReviewPage は正規化済みの条件と本文からページを作り、不正な応答ではゼロ値を返す。
// Body の読み取り・サイズ制限・解放は呼び出し側が担当する。
func decodeReviewPage(appID steamdomain.AppID, options steamdomain.PageOptions, body []byte) (steamdomain.ReviewPage, error) {
	var result reviewPageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return steamdomain.ReviewPage{}, fmt.Errorf("decode Steam review response: %w", errors.Join(ErrInvalidResponse, err))
	}

	if result.Response == nil {
		return steamdomain.ReviewPage{}, fmt.Errorf("missing Steam review response: %w", ErrInvalidResponse)
	}

	if len(result.Response.Reviews) > options.PageSize {
		return steamdomain.ReviewPage{}, fmt.Errorf("Steam review count %d exceeds requested page size %d: %w", len(result.Response.Reviews), options.PageSize, ErrInvalidResponse)
	}

	if len(result.Response.Reviews) > 0 && result.Response.Cursor == "" {
		return steamdomain.ReviewPage{}, fmt.Errorf("missing next cursor: %w", ErrInvalidResponse)
	}

	reviews := make([]steamdomain.Review, len(result.Response.Reviews))
	for i, input := range result.Response.Reviews {
		review, err := convertReview(input)
		if err != nil {
			return steamdomain.ReviewPage{}, fmt.Errorf("convert Steam review %d: %w", i+1, err)
		}

		reviews[i] = review
	}

	return steamdomain.ReviewPage{AppID: appID, Options: options, Reviews: reviews, NextCursor: result.Response.Cursor}, nil
}

// convertReview は受信したレビュー1件の ID と各項目を内部型へ変換する。
// 不正な ID ではゼロ値と ErrInvalidResponse・元の数値変換原因を返す。通信・保存は行わない。
func convertReview(input reviewPageReview) (steamdomain.Review, error) {
	id, err := parseRecommendationID(input.RecommendationID)
	if err != nil {
		return steamdomain.Review{}, fmt.Errorf("invalid Steam recommendationid: %w", errors.Join(ErrInvalidResponse, err))
	}

	return steamdomain.Review{
		RecommendationID: id,
		Text:             input.Text,
		Language:         input.Language,
		Recommended:      input.Recommended,
		TimestampCreated: input.TimestampCreated,
		TimestampUpdated: input.TimestampUpdated,
	}, nil
}
