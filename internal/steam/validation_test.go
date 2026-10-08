package steam_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
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
			client, err := steam.NewClient(tt.client, tt.endpoint)
			if tt.invalid {
				if client != nil || !errors.Is(err, steam.ErrInvalidArgument) {
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
		appID   uint32
		options steam.PageOptions
	}{
		{"AppIDゼロを拒否する", 0, steam.PageOptions{}},
		{"負の件数を拒否する", 440, steam.PageOptions{PageSize: -1}},
		{"101件を拒否する", 440, steam.PageOptions{PageSize: 101}},
		{"大文字の言語を拒否する", 440, steam.PageOptions{Language: "English"}},
		{"言語の空白を拒否する", 440, steam.PageOptions{Language: "en glish"}},
		{"数字で始まる言語を拒否する", 440, steam.PageOptions{Language: "1english"}},
		{"長い言語を拒否する", 440, steam.PageOptions{Language: strings.Repeat("a", 33)}},
		{"長いカーソルを拒否する", 440, steam.PageOptions{Cursor: strings.Repeat("a", 4097)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client, err := steam.NewClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("unexpected HTTP") })}, "")
			if err != nil {
				t.Fatal(err)
			}
			page, err := client.FetchReviewsPage(context.Background(), tt.appID, tt.options)
			if !errors.Is(err, steam.ErrInvalidArgument) || calls != 0 || page.AppID != 0 || page.Reviews != nil {
				t.Fatalf("page=%#v error=%v calls=%d", page, err, calls)
			}
		})
	}
}
