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
	if result.Response == nil || len(result.Response.Reviews) > options.PageSize {
		return steamdomain.ReviewPage{}, fmt.Errorf("missing response or too many reviews: %w", ErrInvalidResponse)
	}
	if len(result.Response.Reviews) > 0 && result.Response.Cursor == "" {
		return steamdomain.ReviewPage{}, fmt.Errorf("missing next cursor: %w", ErrInvalidResponse)
	}
	reviews := make([]steamdomain.Review, len(result.Response.Reviews))
	for i, review := range result.Response.Reviews {
		id, err := parseRecommendationID(review.RecommendationID)
		if err != nil {
			return steamdomain.ReviewPage{}, fmt.Errorf("invalid Steam recommendationid: %w", errors.Join(ErrInvalidResponse, err))
		}
		reviews[i] = steamdomain.Review{RecommendationID: id, Text: review.Text, Language: review.Language, Recommended: review.Recommended, TimestampCreated: review.TimestampCreated, TimestampUpdated: review.TimestampUpdated}
	}
	return steamdomain.ReviewPage{AppID: appID, Options: options, Reviews: reviews, NextCursor: result.Response.Cursor}, nil
}
