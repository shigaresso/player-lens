package steam

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	// Steam API の公式列挙名を保持し、アプリケーションの取得条件と区別する。
	// https://partner.steamgames.com/doc/webapi/IUserReviewsService#Enums
	k_EUserReviewsAppReviewsFilter_Recent = 1
	k_EUserReviewsPurchaseType_All        = 1

	k_EUserReviewsReviewType_All      = 0
	k_EUserReviewsReviewType_Positive = 1
	k_EUserReviewsReviewType_Negative = 2
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
		return k_EUserReviewsReviewType_All, nil
	case ReviewTypePositive:
		return k_EUserReviewsReviewType_Positive, nil
	case ReviewTypeNegative:
		return k_EUserReviewsReviewType_Negative, nil
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
		Filter:         k_EUserReviewsAppReviewsFilter_Recent,
		Languages:      []string{options.Language},
		Cursor:         options.Cursor,
		PageSize:       options.PageSize,
		ReviewType:     reviewType,
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
