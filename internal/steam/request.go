package steam

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	reviewFilterRecent    = 1
	reviewPurchaseTypeAll = 1
)

type reviewPageRequest struct {
	AppID          uint32   `json:"appid"`
	Filter         int      `json:"filter"`
	Languages      []string `json:"languages"`
	Cursor         string   `json:"cursor"`
	PageSize       int      `json:"num_per_page"`
	ReviewType     int      `json:"review_type"`
	PurchaseType   int      `json:"purchase_type"`
	FilterOfftopic bool     `json:"filter_offtopic_activity"`
}

// steamReviewTypeCode は評価種別を Steam API の EUserReviewsReviewType へ変換する。
func steamReviewTypeCode(reviewType ReviewType) (int, error) {
	switch reviewType {
	case ReviewTypeAll:
		return 0, nil
	case ReviewTypePositive:
		return 1, nil
	case ReviewTypeNegative:
		return 2, nil
	default:
		return 0, fmt.Errorf("invalid Steam review type: %w", ErrInvalidArgument)
	}
}

// newReviewPageRequest は正規化済みの取得条件を API の入力データへ変換する。
func newReviewPageRequest(appID uint32, options PageOptions) (reviewPageRequest, error) {
	reviewType, err := steamReviewTypeCode(options.ReviewType)
	if err != nil {
		return reviewPageRequest{}, err
	}
	return reviewPageRequest{
		AppID:          appID,
		Filter:         reviewFilterRecent,
		Languages:      []string{options.Language},
		Cursor:         options.Cursor,
		PageSize:       options.PageSize,
		ReviewType:     reviewType,
		PurchaseType:   reviewPurchaseTypeAll,
		FilterOfftopic: true,
	}, nil
}

func encodeReviewPageQuery(input reviewPageRequest) (url.Values, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encode Steam review request: %w", err)
	}
	return url.Values{"format": {"json"}, "input_json": {string(data)}}, nil
}
