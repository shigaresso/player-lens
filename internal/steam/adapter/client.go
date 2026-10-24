// Package adapter は Steam HTTP API の1ページ取得を担当する。
package adapter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

const defaultEndpoint = "https://api.steampowered.com/IUserReviewsService/GetAppReviews/v1/"
const maxResponseBytes = 2 * 1024 * 1024

// ErrInvalidResponse は JSON または取得結果がページの契約を満たさないことを表す。
var ErrInvalidResponse = errors.New("invalid Steam review response")

// ErrResponseTooLarge は応答が2 MiBを超えたことを表す。
var ErrResponseTooLarge = errors.New("steam review response exceeds 2 MiB")

// HTTPStatusError は200以外の応答を表す。再試行するかどうかは呼び出し側が決める。
type HTTPStatusError struct{ StatusCode int }

// Error は本文を含めず HTTP ステータスを返す。
func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("Steam review HTTP status %d", e.StatusCode)
}

// Client は1回の API 通信を担当する。接続プールの解放は呼び出し側が担当する。
type Client struct {
	httpClient *http.Client
	endpoint   string
}

// NewClient は外から渡された HTTP クライアントと接続先を使う。空の接続先は公式 API となる。
func NewClient(httpClient *http.Client, endpoint string) (*Client, error) {
	if httpClient == nil {
		return nil, fmt.Errorf("HTTP client is required: %w", steamdomain.ErrInvalidArgument)
	}
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return nil, fmt.Errorf("invalid Steam review endpoint: %w", steamdomain.ErrInvalidArgument)
	}
	clientCopy := *httpClient
	clientCopy.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{httpClient: &clientCopy, endpoint: endpoint}, nil
}

// FetchReviewsPage は1ページを取得する。再試行や保存は行わず、失敗時にはゼロ値を返す。
func (c *Client) FetchReviewsPage(ctx context.Context, appID steamdomain.AppID, options steamdomain.PageOptions) (page steamdomain.ReviewPage, err error) {
	if appID == 0 {
		return steamdomain.ReviewPage{}, fmt.Errorf("invalid Steam AppID: %w", steamdomain.ErrInvalidArgument)
	}
	options, err = steamdomain.NormalizePageOptions(options)
	if err != nil {
		return steamdomain.ReviewPage{}, fmt.Errorf("prepare Steam review page options: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	input, err := newReviewPageRequest(appID, options)
	if err != nil {
		return steamdomain.ReviewPage{}, err
	}
	query, err := encodeReviewPageQuery(input)
	if err != nil {
		return steamdomain.ReviewPage{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return steamdomain.ReviewPage{}, fmt.Errorf("create Steam review request: %w", err)
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return steamdomain.ReviewPage{}, fmt.Errorf("fetch Steam reviews: %w", err)
	}
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			page = steamdomain.ReviewPage{}
			err = errors.Join(err, fmt.Errorf("close Steam review response: %w", closeErr))
		}
	}()
	if response.StatusCode != http.StatusOK {
		return steamdomain.ReviewPage{}, &HTTPStatusError{StatusCode: response.StatusCode}
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return steamdomain.ReviewPage{}, fmt.Errorf("read Steam review response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return steamdomain.ReviewPage{}, ErrResponseTooLarge
	}
	return decodeReviewPage(appID, options, body)
}
