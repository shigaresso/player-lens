// Package steam は Steam のレビュー API への HTTP adapter を提供する。
package steam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

const defaultEndpoint = "https://api.steampowered.com/IUserReviewsService/GetAppReviews/v1/"
const maxResponseBytes = 2 * 1024 * 1024

// ErrInvalidArgument は通信前に入力または接続設定が拒否されたことを表す。
var ErrInvalidArgument = errors.New("invalid Steam review argument")

// ErrInvalidResponse は JSON または取得結果がページの契約を満たさないことを表す。
var ErrInvalidResponse = errors.New("invalid Steam review response")

// ErrResponseTooLarge は応答が2 MiBを超えたことを表す。
var ErrResponseTooLarge = errors.New("Steam review response exceeds 2 MiB")

var languagePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

var reviewIDPattern = regexp.MustCompile(`^[0-9]+$`)

// HTTPStatusError は200以外の応答を表す。再試行するかどうかは呼び出し側が決める。
type HTTPStatusError struct{ StatusCode int }

// Error は本文を含めず HTTP ステータスを返す。
func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("Steam review HTTP status %d", e.StatusCode)
}

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
	if httpClient == nil {
		return nil, fmt.Errorf("HTTP client is required: %w", ErrInvalidArgument)
	}
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return nil, fmt.Errorf("invalid Steam review endpoint: %w", ErrInvalidArgument)
	}
	clientCopy := *httpClient
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{httpClient: &clientCopy, endpoint: endpoint}, nil
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
	if appID == 0 || options.PageSize < 1 || options.PageSize > 100 || !languagePattern.MatchString(options.Language) || len(options.Cursor) > 4096 {
		return ReviewPage{}, fmt.Errorf("invalid Steam review page options or AppID: %w", ErrInvalidArgument)
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
		return ReviewPage{}, &HTTPStatusError{StatusCode: response.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return ReviewPage{}, fmt.Errorf("read Steam review response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return ReviewPage{}, ErrResponseTooLarge
	}
	var result struct {
		Response *struct {
			Reviews []Review `json:"reviews"`
			Cursor  string   `json:"cursor"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return ReviewPage{}, fmt.Errorf("decode Steam review response: %w", errors.Join(ErrInvalidResponse, err))
	}
	if result.Response == nil || len(result.Response.Reviews) > options.PageSize {
		return ReviewPage{}, fmt.Errorf("missing response or too many reviews: %w", ErrInvalidResponse)
	}
	if len(result.Response.Reviews) > 0 && result.Response.Cursor == "" {
		return ReviewPage{}, fmt.Errorf("missing next cursor: %w", ErrInvalidResponse)
	}
	for _, review := range result.Response.Reviews {
		id, parseErr := strconv.ParseUint(review.ID, 10, 64)
		if parseErr != nil || id == 0 || !reviewIDPattern.MatchString(review.ID) {
			return ReviewPage{}, fmt.Errorf("invalid review ID: %w", ErrInvalidResponse)
		}
	}
	if result.Response.Reviews == nil {
		result.Response.Reviews = []Review{}
	}
	return ReviewPage{AppID: appID, Options: options, Reviews: result.Response.Reviews, NextCursor: result.Response.Cursor}, nil
}
