package steam

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	// Steam API の公式列挙名を保持し、アプリケーションの取得条件と区別する。
	// https://partner.steamgames.com/doc/webapi/IUserReviewsService#Enums
	k_EUserReviewsAppReviewsFilter_Recent EUserReviewsAppReviewsFilter = 1
	k_EUserReviewsPurchaseType_All        EUserReviewsPurchaseType     = 1
)

type reviewPageRequest struct {
	AppID          AppID                        `json:"appid"`
	Filter         EUserReviewsAppReviewsFilter `json:"filter"`
	Languages      []string                     `json:"languages"`
	Cursor         string                       `json:"cursor"`
	PageSize       int                          `json:"num_per_page"`
	ReviewType     EUserReviewsReviewType       `json:"review_type"`
	PurchaseType   EUserReviewsPurchaseType     `json:"purchase_type"`
	FilterOfftopic bool                         `json:"filter_offtopic_activity"`
}

// newReviewPageRequest は正規化済みの取得条件を API の入力データへ変換する。
func newReviewPageRequest(appID AppID, options PageOptions) (reviewPageRequest, error) {
	switch options.ReviewType {
	case ReviewTypeAll, ReviewTypePositive, ReviewTypeNegative:
	default:
		return reviewPageRequest{}, fmt.Errorf("invalid Steam review type: %w", ErrInvalidArgument)
	}
	return reviewPageRequest{
		AppID:          appID,
		Filter:         k_EUserReviewsAppReviewsFilter_Recent,
		Languages:      []string{options.Language},
		Cursor:         options.Cursor,
		PageSize:       options.PageSize,
		ReviewType:     options.ReviewType,
		PurchaseType:   k_EUserReviewsPurchaseType_All,
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
