package domain_test

import (
	"reflect"
	"slices"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestNewFailedCollection(t *testing.T) {
	for _, tc := range []struct {
		name    string
		reviews []steamdomain.Review
		pages   int
	}{
		{"未取得の nil 配列を保持する", nil, 0},
		{"未取得の空配列を保持する", []steamdomain.Review{}, 0},
		{"部分結果の全項目を保持して配列も複製する", []steamdomain.Review{{RecommendationID: 11, Text: "取得済み", Language: "japanese", Recommended: true, TimestampCreated: 123, TimestampUpdated: 456}}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := steamdomain.Collection{
				AppID: 440,
				Options: steamdomain.CollectionOptionsConfig{
					ReviewType: steamdomain.ReviewTypeNegative, Language: "japanese", PageSize: 10, MaxReviews: 20, MaxPages: 3,
				},
				Reviews: tc.reviews, PagesFetched: tc.pages,
			}
			before := input
			before.Reviews = slices.Clone(input.Reviews)
			want := before
			want.StopReason = steamdomain.StopFailed

			got := steamdomain.NewFailedCollection(input)
			if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(input, before) {
				t.Fatalf("got=%#v input=%#v; want=%#v before=%#v", got, input, want, before)
			}

			if len(got.Reviews) > 0 {
				got.Reviews[0].Text = "返却後の変更"
				if !reflect.DeepEqual(input, before) {
					t.Fatal("失敗結果の変更が入力のレビューへ影響した")
				}
			}
		})
	}
}
