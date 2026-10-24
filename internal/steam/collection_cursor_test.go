package steam_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestCollectReviewsCursorCycle(t *testing.T) {
	cases := []struct {
		name   string
		pages  []steam.ReviewPage
		limit  int
		want   []steam.Review
		reason steam.StopReason
		failed bool
	}{
		{"先頭カーソルへの循環を止める", []steam.ReviewPage{{Reviews: collectionReviews(1), NextCursor: "*"}}, 10, collectionReviews(1), steam.StopFailed, true},
		{"同じ次カーソルへの循環を止める", []steam.ReviewPage{{Reviews: collectionReviews(1), NextCursor: "A"}, {Reviews: collectionReviews(2), NextCursor: "A"}}, 10, collectionReviews(1, 2), steam.StopFailed, true},
		{"以前のカーソルへの循環をページ上限より優先する", []steam.ReviewPage{{Reviews: collectionReviews(1), NextCursor: "A"}, {Reviews: collectionReviews(2), NextCursor: "B"}, {Reviews: collectionReviews(3), NextCursor: "A"}}, 10, collectionReviews(1, 2, 3), steam.StopFailed, true},
		{"必要な件数を満たしたら続きの循環を調べない", []steam.ReviewPage{{Reviews: collectionReviews(1), NextCursor: "*"}}, 1, collectionReviews(1), steam.StopReviewLimit, false},
		{"空ページなら続きの循環を調べない", []steam.ReviewPage{{NextCursor: "*"}}, 10, []steam.Review{}, steam.StopExhausted, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			fetcher := pageFetcherFunc(func(_ context.Context, appID steam.AppID, options steam.PageOptions) (steam.ReviewPage, error) {
				if calls >= len(tt.pages) {
					t.Fatal("循環の検出後に取得した")
				}
				page := tt.pages[calls]
				calls++
				page.AppID, page.Options = appID, options
				return page, nil
			})
			got, err := steam.CollectReviews(context.Background(), fetcher, 440, steam.CollectionOptions{PageSize: 1, MaxReviews: tt.limit, MaxPages: len(tt.pages)})
			if (err != nil) != tt.failed || got.StopReason != tt.reason || got.PagesFetched != len(tt.pages) || calls != len(tt.pages) || !reflect.DeepEqual(got.Reviews, tt.want) {
				t.Fatalf("result=%#v calls=%d error=%v", got, calls, err)
			}
		})
	}
}
