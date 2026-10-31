package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

const (
	// Steam API の公式列挙名を保持し、アプリケーションの取得条件と区別する。
	// https://partner.steamgames.com/doc/webapi/IUserReviewsService#Enums
	k_EUserReviewsAppReviewsFilter_Recent steamdomain.EUserReviewsAppReviewsFilter = 1
	k_EUserReviewsPurchaseType_All        steamdomain.EUserReviewsPurchaseType     = 1
)

// reviewPageAPIInput は Steam API の input_json に格納する入力データを表す。
type reviewPageAPIInput struct {
	AppID          steamdomain.AppID                        `json:"appid"`
	Filter         steamdomain.EUserReviewsAppReviewsFilter `json:"filter"`
	Languages      []string                                 `json:"languages"`
	Cursor         string                                   `json:"cursor"`
	PageSize       int                                      `json:"num_per_page"`
	ReviewType     steamdomain.EUserReviewsReviewType       `json:"review_type"`
	PurchaseType   steamdomain.EUserReviewsPurchaseType     `json:"purchase_type"`
	FilterOfftopic bool                                     `json:"filter_offtopic_activity"`
}

// newReviewPageHTTPRequest は API の入力とクエリを組み立て、context 付きの GET リクエストを作る。
// 呼び出し側は接続先と App ID の検査・取得条件の正規化を完了させ、context の寿命と送信を管理する。
func newReviewPageHTTPRequest(ctx context.Context, endpoint string, appID steamdomain.AppID, options steamdomain.PageOptions) (*http.Request, error) {
	input := newReviewPageAPIInput(appID, options)
	query, err := buildReviewPageQuery(input)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create Steam review request: %w", err)
	}

	return request, nil
}

// newReviewPageAPIInput は正の App ID と正規化済みの取得条件を API の入力データへ変換する。
// 呼び出し側は App ID の検査と NormalizePageOptions を先に完了させる。
func newReviewPageAPIInput(appID steamdomain.AppID, options steamdomain.PageOptions) reviewPageAPIInput {
	return reviewPageAPIInput{
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

// buildReviewPageQuery は API 入力を JSON 化し、format と input_json のクエリパラメーターを返す。
// URL エンコードは戻り値の Encode で行う。
func buildReviewPageQuery(input reviewPageAPIInput) (url.Values, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("encode Steam review request: %w", err)
	}

	return url.Values{"format": {"json"}, "input_json": {string(data)}}, nil
}
