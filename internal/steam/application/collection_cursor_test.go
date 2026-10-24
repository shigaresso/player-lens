package application_test

import (
	"context"
	"reflect"
	"testing"

	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectReviewsCursorCycle(t *testing.T) {
	cases := []struct {
		name   string
		pages  []steamdomain.ReviewPage
		limit  int
		want   []steamdomain.Review
		reason steamdomain.StopReason
		failed bool
	}{
		{"先頭カーソルへの循環を止める", []steamdomain.ReviewPage{{Reviews: collectionReviews(1), NextCursor: "*"}}, 10, collectionReviews(1), steamdomain.StopFailed, true},
		{"同じ次カーソルへの循環を止める", []steamdomain.ReviewPage{{Reviews: collectionReviews(1), NextCursor: "A"}, {Reviews: collectionReviews(2), NextCursor: "A"}}, 10, collectionReviews(1, 2), steamdomain.StopFailed, true},
		{"以前のカーソルへの循環をページ上限より優先する", []steamdomain.ReviewPage{{Reviews: collectionReviews(1), NextCursor: "A"}, {Reviews: collectionReviews(2), NextCursor: "B"}, {Reviews: collectionReviews(3), NextCursor: "A"}}, 10, collectionReviews(1, 2, 3), steamdomain.StopFailed, true},
		{"必要な件数を満たしたら続きの循環を調べない", []steamdomain.ReviewPage{{Reviews: collectionReviews(1), NextCursor: "*"}}, 1, collectionReviews(1), steamdomain.StopReviewLimit, false},
		{"空ページなら続きの循環を調べない", []steamdomain.ReviewPage{{NextCursor: "*"}}, 10, []steamdomain.Review{}, steamdomain.StopExhausted, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			fetcher := pageFetcherFunc(func(_ context.Context, appID steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				if calls >= len(tt.pages) {
					t.Fatal("循環の検出後に取得した")
				}
				page := tt.pages[calls]
				calls++
				page.AppID, page.Options = appID, options
				return page, nil
			})
			got, err := steamapplication.CollectReviews(context.Background(), fetcher, 440, steamdomain.CollectionOptions{PageSize: 1, MaxReviews: tt.limit, MaxPages: len(tt.pages)})
			if (err != nil) != tt.failed || got.StopReason != tt.reason || got.PagesFetched != len(tt.pages) || calls != len(tt.pages) || !reflect.DeepEqual(got.Reviews, tt.want) {
				t.Fatalf("result=%#v calls=%d error=%v", got, calls, err)
			}
		})
	}
}
