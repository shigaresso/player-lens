package adapter

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestNewReviewPageRequest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options steamdomain.PageOptions
		want    reviewPageRequest
		invalid bool
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
			name:    "未知の評価種別ではゼロ値と入力エラーを返す",
			options: steamdomain.PageOptions{ReviewType: 3, Language: "japanese", PageSize: 7},
			invalid: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := newReviewPageRequest(440, tc.options)
			if !reflect.DeepEqual(got, tc.want) || (err != nil) != tc.invalid || errors.Is(err, steamdomain.ErrInvalidArgument) != tc.invalid {
				t.Fatalf("request=%+v error=%v want=%+v invalid=%v", got, err, tc.want, tc.invalid)
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
