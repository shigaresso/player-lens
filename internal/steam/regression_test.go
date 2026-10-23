package steam_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestFetchReviewsPageValidBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name    string
		appID   steam.AppID
		options steam.PageOptions
	}{
		{"最小のAppIDと件数を受け付ける", 1, steam.PageOptions{Language: "all", Cursor: steam.FirstPageCursor, PageSize: 1}},
		{"最大のAppIDと入力長を受け付ける", ^steam.AppID(0), steam.PageOptions{Language: "a" + strings.Repeat("b", 31), Cursor: strings.Repeat("x", 4096), PageSize: 100}},
		{"言語の数字と区切りを受け付ける", 440, steam.PageOptions{Language: "en_1-test", Cursor: steam.FirstPageCursor, PageSize: 20}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tt.options.ReviewType = steam.ReviewTypeAll
			calls := 0
			client, err := steam.NewClient(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Scheme != "https" || r.URL.Host != "api.steampowered.com" || r.URL.Path != "/IUserReviewsService/GetAppReviews/v1/" || len(r.URL.Query()) != 2 {
					t.Errorf("URL=%s", r.URL)
				}
				var input struct {
					AppID     steam.AppID `json:"appid"`
					Languages []string    `json:"languages"`
					Cursor    string      `json:"cursor"`
					PageSize  int         `json:"num_per_page"`
				}
				if err := json.Unmarshal([]byte(r.URL.Query().Get("input_json")), &input); err != nil {
					t.Error(err)
				}
				if input.AppID != tt.appID || input.Cursor != tt.options.Cursor || input.PageSize != tt.options.PageSize || !reflect.DeepEqual(input.Languages, []string{tt.options.Language}) {
					t.Errorf("input=%#v", input)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"response":{}}`)), Header: make(http.Header)}, nil
			})}, "")
			if err != nil {
				t.Fatal(err)
			}
			page, err := client.FetchReviewsPage(context.Background(), tt.appID, tt.options)
			if err != nil || page.AppID != tt.appID || page.Options != tt.options || calls != 1 {
				t.Fatalf("page=%#v error=%v calls=%d", page, err, calls)
			}
		})
	}
}

func TestFetchReviewsPageTransportFailure(t *testing.T) {
	failure := errors.New("transport failed")
	calls := 0
	client, err := steam.NewClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, failure })}, "")
	if err != nil {
		t.Fatal(err)
	}
	page, err := client.FetchReviewsPage(context.Background(), 440, steam.PageOptions{})
	if !errors.Is(err, failure) || calls != 1 || !reflect.DeepEqual(page, steam.ReviewPage{}) {
		t.Fatalf("page=%#v error=%v calls=%d", page, err, calls)
	}
}
