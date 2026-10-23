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

	k_EUserReviewsReviewType_All      EUserReviewsReviewType = 0
	k_EUserReviewsReviewType_Positive EUserReviewsReviewType = 1
	k_EUserReviewsReviewType_Negative EUserReviewsReviewType = 2
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

// steamReviewTypeCode は評価種別を Steam API の EUserReviewsReviewType へ変換する。
func steamReviewTypeCode(reviewType ReviewType) (EUserReviewsReviewType, error) {
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
func newReviewPageRequest(appID AppID, options PageOptions) (reviewPageRequest, error) {
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
