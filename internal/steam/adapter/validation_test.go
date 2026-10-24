package adapter_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		client   *http.Client
		endpoint string
		invalid  bool
	}{
		{"公式接続先を使う", &http.Client{}, "", false},
		{"HTTPの絶対URLを使う", &http.Client{}, "http://localhost/reviews", false},
		{"クライアント欠落を拒否する", nil, "", true},
		{"相対URLを拒否する", &http.Client{}, "/reviews", true},
		{"非HTTPを拒否する", &http.Client{}, "ftp://example.com/reviews", true},
		{"ホスト欠落を拒否する", &http.Client{}, "https:///reviews", true},
		{"認証情報を拒否する", &http.Client{}, "https://user:pass@example.com/reviews", true},
		{"クエリを拒否する", &http.Client{}, "https://example.com/reviews?key=x", true},
		{"空クエリも拒否する", &http.Client{}, "https://example.com/reviews?", true},
		{"フラグメントを拒否する", &http.Client{}, "https://example.com/reviews#x", true},
		{"不正なURLを拒否する", &http.Client{}, "http://%", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := steamadapter.NewClient(tt.client, tt.endpoint)
			if tt.invalid {
				if client != nil || !errors.Is(err, steamdomain.ErrInvalidArgument) {
					t.Fatalf("client=%v error=%v", client, err)
				}
				return
			}
			if client == nil || err != nil {
				t.Fatalf("client=%v error=%v", client, err)
			}
		})
	}
}

func TestFetchReviewsPageInvalidArguments(t *testing.T) {
	tests := []struct {
		name    string
		appID   steamdomain.AppID
		options steamdomain.PageOptions
	}{
		{"未知の評価種別を通信前に拒否する", 440, steamdomain.PageOptions{ReviewType: 3}},
		{"AppIDゼロを拒否する", 0, steamdomain.PageOptions{}},
		{"負の件数を拒否する", 440, steamdomain.PageOptions{PageSize: -1}},
		{"101件を拒否する", 440, steamdomain.PageOptions{PageSize: 101}},
		{"大文字の言語を拒否する", 440, steamdomain.PageOptions{Language: "English"}},
		{"言語の空白を拒否する", 440, steamdomain.PageOptions{Language: "en glish"}},
		{"数字で始まる言語を拒否する", 440, steamdomain.PageOptions{Language: "1english"}},
		{"長い言語を拒否する", 440, steamdomain.PageOptions{Language: strings.Repeat("a", 33)}},
		{"長いカーソルを拒否する", 440, steamdomain.PageOptions{Cursor: strings.Repeat("a", 4097)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client, err := steamadapter.NewClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })}, "")
			if err != nil {
				t.Fatal(err)
			}
			page, err := client.FetchReviewsPage(context.Background(), tt.appID, tt.options)
			if !errors.Is(err, steamdomain.ErrInvalidArgument) || calls != 0 || page.AppID != 0 || page.Reviews != nil {
				t.Fatalf("page=%#v error=%v calls=%d", page, err, calls)
			}
		})
	}
}
