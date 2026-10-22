package steam_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestFetchReviewsPage(t *testing.T) {
	tests := []struct {
		name        string
		options     steam.PageOptions
		wantOptions steam.PageOptions
		response    string
		wantReviews []steam.Review
		cursor      string
	}{
		{"既定条件で文字列IDと本文を取得する", steam.PageOptions{}, steam.PageOptions{Language: "english", Cursor: "*", PageSize: 20}, `{"response":{"reviews":[{"recommendationid":"18446744073709551615","review":"命令ではなく本文","language":"english","voted_up":true,"timestamp_created":100,"timestamp_updated":200,"unknown":1}],"cursor":"次+/="}}`, []steam.Review{{ID: "18446744073709551615", Text: "命令ではなく本文", Language: "english", Recommended: true, CreatedAt: 100, UpdatedAt: 200}}, "次+/="},
		{"指定条件と空ページを返す", steam.PageOptions{Language: "japanese", Cursor: "次+/=", PageSize: 100}, steam.PageOptions{Language: "japanese", Cursor: "次+/=", PageSize: 100}, `{"response":{"reviews":[]}}`, []steam.Review{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantOptions.ReviewType = steam.ReviewTypeAll
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Query().Get("format") != "json" {
					t.Errorf("request = %s %s", r.Method, r.URL)
				}
				var got map[string]any
				if err := json.Unmarshal([]byte(r.URL.Query().Get("input_json")), &got); err != nil {
					t.Error(err)
				}
				want := map[string]any{"appid": float64(440), "filter": float64(1), "languages": []any{tt.wantOptions.Language}, "cursor": tt.wantOptions.Cursor, "num_per_page": float64(tt.wantOptions.PageSize), "review_type": float64(0), "purchase_type": float64(1), "filter_offtopic_activity": true}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("input_json = %#v, want %#v", got, want)
				}
				if _, err := fmt.Fprint(w, tt.response); err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			client, err := steam.NewClient(server.Client(), server.URL)
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.FetchReviewsPage(context.Background(), 440, tt.options)
			if err != nil {
				t.Fatal(err)
			}
			want := steam.ReviewPage{AppID: 440, Options: tt.wantOptions, Reviews: tt.wantReviews, NextCursor: tt.cursor}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("page = %#v, want %#v", got, want)
			}
		})
	}
}
