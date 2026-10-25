package adapter

import (
	"encoding/json"
	"reflect"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestNewReviewPageRequest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options steamdomain.PageOptions
		want    reviewPageRequest
	}{
		{
			name:    "取得条件を保持し固定のSteam取得条件を設定する",
			options: steamdomain.PageOptions{ReviewType: steamdomain.ReviewTypeNegative, Language: "japanese", Cursor: "次+/=&?", PageSize: 7},
			want: reviewPageRequest{
				AppID: 440, Filter: 1, Languages: []string{"japanese"},
				Cursor: "次+/=&?", PageSize: 7, ReviewType: 2, PurchaseType: 1, FilterOfftopic: true,
			},
		},
		{
			name:    "全評価をAPIの全評価へ変換する",
			options: steamdomain.PageOptions{ReviewType: steamdomain.ReviewTypeAll, Language: "japanese", Cursor: "次+/=&?", PageSize: 7},
			want: reviewPageRequest{
				AppID: 440, Filter: 1, Languages: []string{"japanese"},
				Cursor: "次+/=&?", PageSize: 7, ReviewType: 0, PurchaseType: 1, FilterOfftopic: true,
			},
		},
		{
			name:    "おすすめをAPIのおすすめへ変換する",
			options: steamdomain.PageOptions{ReviewType: steamdomain.ReviewTypePositive, Language: "japanese", Cursor: "次+/=&?", PageSize: 7},
			want: reviewPageRequest{
				AppID: 440, Filter: 1, Languages: []string{"japanese"},
				Cursor: "次+/=&?", PageSize: 7, ReviewType: 1, PurchaseType: 1, FilterOfftopic: true,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := newReviewPageRequest(440, tc.options)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("request=%+v want=%+v", got, tc.want)
			}
		})
	}
}

func TestEncodeReviewPageQuery(t *testing.T) {
	t.Run("APIの項目名でJSON化し言語と特殊文字のカーソルを保持する", func(t *testing.T) {
		input := reviewPageRequest{
			AppID: 440, Filter: 1, Languages: []string{"japanese"},
			Cursor: "次+/=&?", PageSize: 7, ReviewType: 2, PurchaseType: 1, FilterOfftopic: true,
		}
		query, err := encodeReviewPageQuery(input)
		if err != nil {
			t.Fatal(err)
		}

		if len(query) != 2 || !reflect.DeepEqual(query["format"], []string{"json"}) || len(query["input_json"]) != 1 {
			t.Fatalf("query=%v", query)
		}

		var got map[string]any
		if err := json.Unmarshal([]byte(query.Get("input_json")), &got); err != nil {
			t.Fatal(err)
		}

		want := map[string]any{
			"appid": float64(440), "filter": float64(1), "languages": []any{"japanese"},
			"cursor": "次+/=&?", "num_per_page": float64(7), "review_type": float64(2),
			"purchase_type": float64(1), "filter_offtopic_activity": true,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("input_json=%#v want=%#v", got, want)
		}
	})
}
