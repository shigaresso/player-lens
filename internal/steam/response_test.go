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

func clientWithResponse(t *testing.T, body io.ReadCloser) *steam.Client {
	t.Helper()
	client, err := steam.NewClient(&http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: body, Header: make(http.Header)}, nil
	})}, "")
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestFetchReviewsPageInvalidResponse(t *testing.T) {
	tests := []struct{ name, body string }{
		{"JSON構文の不正を拒否する", `{"response":`},
		{"後続のJSONを拒否する", `{"response":{}}{}`},
		{"wrapper欠落を拒否する", `{}`},
		{"null応答を拒否する", `{"response":null}`},
		{"配列のwrapperを拒否する", `{"response":[]}`},
		{"数値のIDを拒否する", `{"response":{"reviews":[{"recommendationid":1}],"cursor":"x"}}`},
		{"空のIDを拒否する", `{"response":{"reviews":[{}],"cursor":"x"}}`},
		{"ゼロIDを拒否する", `{"response":{"reviews":[{"recommendationid":"0"}],"cursor":"x"}}`},
		{"負のIDを拒否する", `{"response":{"reviews":[{"recommendationid":"-1"}],"cursor":"x"}}`},
		{"符号付きIDを拒否する", `{"response":{"reviews":[{"recommendationid":"+1"}],"cursor":"x"}}`},
		{"uint64を超えるIDを拒否する", `{"response":{"reviews":[{"recommendationid":"18446744073709551616"}],"cursor":"x"}}`},
		{"非空ページのカーソル欠落を拒否する", `{"response":{"reviews":[{"recommendationid":"1"}]}}`},
		{"要求件数を超える配列を拒否する", `{"response":{"reviews":[{"recommendationid":"1"},{"recommendationid":"2"}],"cursor":"x"}}`},
		{"時刻の型違いを拒否する", `{"response":{"reviews":[{"recommendationid":"1","timestamp_created":"100"}],"cursor":"x"}}`},
		{"時刻の範囲外を拒否する", `{"response":{"reviews":[{"recommendationid":"1","timestamp_updated":4294967296}],"cursor":"x"}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := clientWithResponse(t, io.NopCloser(strings.NewReader(tt.body)))
			page, err := client.FetchReviewsPage(context.Background(), 440, steam.PageOptions{PageSize: 1})
			if !errors.Is(err, steam.ErrInvalidResponse) || !reflect.DeepEqual(page, steam.ReviewPage{}) {
				t.Fatalf("page=%#v error=%v", page, err)
			}
		})
	}
}

func TestFetchReviewsPageJSONErrorCause(t *testing.T) {
	client := clientWithResponse(t, io.NopCloser(strings.NewReader(`{"response":`)))
	_, err := client.FetchReviewsPage(context.Background(), 440, steam.PageOptions{})
	var syntaxError *json.SyntaxError
	if !errors.As(err, &syntaxError) {
		t.Fatalf("JSONの原因を失った: %v", err)
	}
}

func TestFetchReviewsPageResponseSize(t *testing.T) {
	for _, tt := range []struct {
		name    string
		size    int
		invalid bool
	}{
		{"2MiBちょうどは取得できる", 2 * 1024 * 1024, false},
		{"2MiB超過を拒否する", 2*1024*1024 + 1, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			prefix := `{"response":{}}`
			body := prefix + strings.Repeat(" ", tt.size-len(prefix))
			client := clientWithResponse(t, io.NopCloser(strings.NewReader(body)))
			page, err := client.FetchReviewsPage(context.Background(), 440, steam.PageOptions{})
			if tt.invalid {
				if err == nil || !reflect.DeepEqual(page, steam.ReviewPage{}) {
					t.Fatalf("page=%#v error=%v", page, err)
				}
				return
			}
			if err != nil || page.AppID != 440 || len(page.Reviews) != 0 {
				t.Fatalf("page=%#v error=%v", page, err)
			}
		})
	}
}
