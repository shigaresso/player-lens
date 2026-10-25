package adapter

import (
	"encoding/json"
	"fmt"
	"net/url"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

const (
	// Steam API の公式列挙名を保持し、アプリケーションの取得条件と区別する。
	// https://partner.steamgames.com/doc/webapi/IUserReviewsService#Enums
	k_EUserReviewsAppReviewsFilter_Recent steamdomain.EUserReviewsAppReviewsFilter = 1
	k_EUserReviewsPurchaseType_All        steamdomain.EUserReviewsPurchaseType     = 1
)

type reviewPageRequest struct {
	AppID          steamdomain.AppID                        `json:"appid"`
	Filter         steamdomain.EUserReviewsAppReviewsFilter `json:"filter"`
	Languages      []string                                 `json:"languages"`
	Cursor         string                                   `json:"cursor"`
	PageSize       int                                      `json:"num_per_page"`
	ReviewType     steamdomain.EUserReviewsReviewType       `json:"review_type"`
	PurchaseType   steamdomain.EUserReviewsPurchaseType     `json:"purchase_type"`
	FilterOfftopic bool                                     `json:"filter_offtopic_activity"`
}

// newReviewPageRequest は正の App ID と正規化済みの取得条件を API の入力データへ変換する。
// 呼び出し側は App ID の検査と NormalizePageOptions を先に完了させる。
func newReviewPageRequest(appID steamdomain.AppID, options steamdomain.PageOptions) reviewPageRequest {
	return reviewPageRequest{
		AppID:          appID,
		Filter:         k_EUserReviewsAppReviewsFilter_Recent,
		Languages:      []string{options.Language},
		Cursor:         options.Cursor,
		PageSize:       options.PageSize,
		ReviewType:     steamdomain.EUserReviewsReviewType(options.ReviewType),
		PurchaseType:   k_EUserReviewsPurchaseType_All,
		FilterOfftopic: true,
	}
}

func encodeReviewPageQuery(input reviewPageRequest) (url.Values, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encode Steam review request: %w", err)
	}
	return url.Values{"format": {"json"}, "input_json": {string(data)}}, nil
}
