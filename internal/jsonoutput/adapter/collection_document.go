package adapter

import (
	"fmt"
	"strconv"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type collectionDocument struct {
	SchemaVersion int                    `json:"schema_version"`
	AppID         steamdomain.AppID      `json:"app_id"`
	Options       collectionOptions      `json:"options"`
	Reviews       []reviewDocument       `json:"reviews"`
	PagesFetched  int                    `json:"pages_fetched"`
	StopReason    steamdomain.StopReason `json:"stop_reason"`
}

type collectionOptions struct {
	ReviewType string `json:"review_type"`
	Language   string `json:"language"`
	PageSize   int    `json:"page_size"`
	MaxReviews int    `json:"max_reviews"`
	MaxPages   int    `json:"max_pages"`
}

type reviewDocument struct {
	RecommendationID string                       `json:"id"`
	Text             string                       `json:"text"`
	Language         string                       `json:"language"`
	Recommended      bool                         `json:"recommended"`
	TimestampCreated steamdomain.TimestampCreated `json:"created_at"`
	TimestampUpdated steamdomain.TimestampUpdated `json:"updated_at"`
}

// newCollectionDocument は出力形式への変換と構築を行い、入力を変更しない。
// 未知の評価種別では出力データのゼロ値と原因を保持したエラーを返す。
func newCollectionDocument(result steamdomain.Collection) (collectionDocument, error) {
	reviewType, err := steamdomain.FormatReviewType(result.Options.ReviewType)
	if err != nil {
		return collectionDocument{}, fmt.Errorf("prepare review collection JSON: %w", err)
	}

	reviews := make([]reviewDocument, len(result.Reviews))
	for i, review := range result.Reviews {
		reviews[i] = reviewDocument{
			RecommendationID: strconv.FormatUint(uint64(review.RecommendationID), 10),
			Text:             review.Text,
			Language:         review.Language,
			Recommended:      review.Recommended,
			TimestampCreated: review.TimestampCreated,
			TimestampUpdated: review.TimestampUpdated,
		}
	}

	return collectionDocument{
		SchemaVersion: 1,
		AppID:         result.AppID,
		Options: collectionOptions{
			ReviewType: reviewType,
			Language:   result.Options.Language,
			PageSize:   result.Options.PageSize,
			MaxReviews: result.Options.MaxReviews,
			MaxPages:   result.Options.MaxPages,
		},
		Reviews:      reviews,
		PagesFetched: result.PagesFetched,
		StopReason:   result.StopReason,
	}, nil
}
