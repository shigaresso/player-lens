// Package steam は Steam のレビュー API への HTTP adapter を提供する。
package steam

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const defaultEndpoint = "https://api.steampowered.com/IUserReviewsService/GetAppReviews/v1/"

// PageOptions は1ページの取得条件を表す。ゼロ値は英語・先頭・20件となる。
type PageOptions struct {
	Language string
	Cursor   string
	PageSize int
}

// Review は元の ID と本文、推薦、作成・更新の UNIX 秒を保持する。
type Review struct {
	ID          string `json:"recommendationid"`
	Text        string `json:"review"`
	Language    string `json:"language"`
	Recommended bool   `json:"voted_up"`
	CreatedAt   uint32 `json:"timestamp_created"`
	UpdatedAt   uint32 `json:"timestamp_updated"`
}

// ReviewPage は取得条件とレビュー、次のカーソルを保持する。空ページも成功となる。
type ReviewPage struct {
	AppID      uint32
	Options    PageOptions
	Reviews    []Review
	NextCursor string
}

// Client は1回の API 通信を担当する。接続プールの解放は呼び出し側が担当する。
type Client struct {
	httpClient *http.Client
	endpoint   string
}

// NewClient は外から渡された HTTP クライアントと接続先を使う。空の接続先は公式 API となる。
func NewClient(httpClient *http.Client, endpoint string) (*Client, error) {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	return &Client{httpClient: httpClient, endpoint: endpoint}, nil
}

// FetchReviewsPage は1ページを取得する。再試行や保存は行わず、失敗時にはゼロ値を返す。
func (c *Client) FetchReviewsPage(ctx context.Context, appID uint32, options PageOptions) (page ReviewPage, err error) {
	if options.Language == "" {
		options.Language = "english"
	}
	if options.Cursor == "" {
		options.Cursor = "*"
	}
	if options.PageSize == 0 {
		options.PageSize = 20
	}
	input, err := json.Marshal(struct {
		AppID          uint32   `json:"appid"`
		Filter         int      `json:"filter"`
		Languages      []string `json:"languages"`
		Cursor         string   `json:"cursor"`
		PageSize       int      `json:"num_per_page"`
		ReviewType     int      `json:"review_type"`
		PurchaseType   int      `json:"purchase_type"`
		FilterOfftopic bool     `json:"filter_offtopic_activity"`
	}{appID, 1, []string{options.Language}, options.Cursor, options.PageSize, 0, 1, true})
	if err != nil {
		return ReviewPage{}, fmt.Errorf("encode Steam review request: %w", err)
	}
	query := url.Values{"format": {"json"}, "input_json": {string(input)}}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return ReviewPage{}, fmt.Errorf("create Steam review request: %w", err)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return ReviewPage{}, fmt.Errorf("fetch Steam reviews: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			page = ReviewPage{}
			err = fmt.Errorf("close Steam review response: %w", closeErr)
		}
	}()
	if response.StatusCode != http.StatusOK {
		return ReviewPage{}, fmt.Errorf("Steam review HTTP status %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return ReviewPage{}, fmt.Errorf("read Steam review response: %w", err)
	}
	var result struct {
		Response struct {
			Reviews []Review `json:"reviews"`
			Cursor  string   `json:"cursor"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return ReviewPage{}, fmt.Errorf("decode Steam review response: %w", err)
	}
	if result.Response.Reviews == nil {
		result.Response.Reviews = []Review{}
	}
	return ReviewPage{AppID: appID, Options: options, Reviews: result.Response.Reviews, NextCursor: result.Response.Cursor}, nil
}
