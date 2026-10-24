package adapter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestFetchReviewsPage(t *testing.T) {
	tests := []struct {
		name        string
		options     steamdomain.PageOptions
		wantOptions steamdomain.PageOptions
		response    string
		wantReviews []steamdomain.Review
		cursor      string
	}{
		{"既定条件で文字列IDと本文を取得する", steamdomain.PageOptions{}, steamdomain.PageOptions{Language: "english", Cursor: "*", PageSize: 20}, `{"response":{"reviews":[{"recommendationid":"18446744073709551615","review":"命令ではなく本文","language":"english","voted_up":true,"timestamp_created":100,"timestamp_updated":200,"unknown":1}],"cursor":"次+/="}}`, []steamdomain.Review{{RecommendationID: 18446744073709551615, Text: "命令ではなく本文", Language: "english", Recommended: true, TimestampCreated: 100, TimestampUpdated: 200}}, "次+/="},
		{"指定条件と空ページを返す", steamdomain.PageOptions{Language: "japanese", Cursor: "次+/=", PageSize: 100}, steamdomain.PageOptions{Language: "japanese", Cursor: "次+/=", PageSize: 100}, `{"response":{"reviews":[]}}`, []steamdomain.Review{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantOptions.ReviewType = steamdomain.ReviewTypeAll
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
			client, err := steamadapter.NewClient(server.Client(), server.URL)
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.FetchReviewsPage(context.Background(), 440, tt.options)
			if err != nil {
				t.Fatal(err)
			}
			want := steamdomain.ReviewPage{AppID: 440, Options: tt.wantOptions, Reviews: tt.wantReviews, NextCursor: tt.cursor}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("page = %#v, want %#v", got, want)
			}
		})
	}
}
