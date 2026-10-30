package domain_test

import (
	"errors"
	"reflect"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectionStateCursor(t *testing.T) {
	for _, tc := range []struct {
		name   string
		pages  []steamdomain.ReviewPage
		limit  int
		want   []steamdomain.Review
		reason steamdomain.StopReason
		failed bool
	}{
		{"先頭カーソルへの循環を検出しページの結果を保持する", []steamdomain.ReviewPage{{Reviews: stateReviews(1), NextCursor: "*"}}, 10, stateReviews(1), steamdomain.StopFailed, true},
		{"同じ次カーソルを再取得せず終了する", []steamdomain.ReviewPage{{Reviews: stateReviews(1), NextCursor: "A"}, {Reviews: stateReviews(2), NextCursor: "A"}}, 10, stateReviews(1, 2), steamdomain.StopFailed, true},
		{"以前のカーソルへの循環をページ上限より優先する", []steamdomain.ReviewPage{{Reviews: stateReviews(1), NextCursor: "A"}, {Reviews: stateReviews(2), NextCursor: "B"}, {Reviews: stateReviews(3), NextCursor: "A"}}, 10, stateReviews(1, 2, 3), steamdomain.StopFailed, true},
		{"重複だけのページでも循環を検出する", []steamdomain.ReviewPage{{Reviews: stateReviews(1), NextCursor: "A"}, {Reviews: stateReviews(1), NextCursor: "A"}}, 10, stateReviews(1), steamdomain.StopFailed, true},
		{"件数上限を循環より優先する", []steamdomain.ReviewPage{{Reviews: stateReviews(1), NextCursor: "*"}}, 1, stateReviews(1), steamdomain.StopReviewLimit, false},
		{"空ページを循環より優先する", []steamdomain.ReviewPage{{NextCursor: "*"}}, 10, []steamdomain.Review{}, steamdomain.StopExhausted, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := steamdomain.NewCollectionState(440, collectionOptions(t, steamdomain.CollectionOptionsConfig{MaxReviews: tc.limit, MaxPages: len(tc.pages)}))
			if err != nil {
				t.Fatal(err)
			}

			for i, page := range tc.pages {
				err = state.AddPage(page)
				if i < len(tc.pages)-1 && err != nil {
					t.Fatal(err)
				}
			}

			result := state.Result()
			if (err != nil) != tc.failed || (tc.failed && !errors.Is(err, steamdomain.ErrCursorCycle)) || result.PagesFetched != len(tc.pages) || result.StopReason != tc.reason || !reflect.DeepEqual(result.Reviews, tc.want) {
				t.Fatalf("result=%#v err=%v", result, err)
			}

			if next, ok := state.NextFetchConditions(); ok || next != (steamdomain.PageOptions{}) {
				t.Fatalf("終了後の next=%#v available=%v", next, ok)
			}
		})
	}
}
