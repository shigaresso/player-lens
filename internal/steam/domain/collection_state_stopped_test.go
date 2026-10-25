package domain_test

import (
	"errors"
	"reflect"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectionStateStopped(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options steamdomain.CollectionOptionsConfig
		page    steamdomain.ReviewPage
		reason  steamdomain.StopReason
		fail    bool
		cycle   bool
	}{
		{"空ページで終了した状態を変更しない", steamdomain.CollectionOptionsConfig{}, steamdomain.ReviewPage{}, steamdomain.StopExhausted, false, false},
		{"件数上限で終了した状態を変更しない", steamdomain.CollectionOptionsConfig{MaxReviews: 1}, steamdomain.ReviewPage{Reviews: stateReviews(1), NextCursor: "A"}, steamdomain.StopReviewLimit, false, false},
		{"ページ上限で終了した状態を変更しない", steamdomain.CollectionOptionsConfig{MaxPages: 1}, steamdomain.ReviewPage{Reviews: stateReviews(1), NextCursor: "A"}, steamdomain.StopPageLimit, false, false},
		{"循環で失敗した状態を変更しない", steamdomain.CollectionOptionsConfig{}, steamdomain.ReviewPage{Reviews: stateReviews(1), NextCursor: "*"}, steamdomain.StopFailed, false, true},
		{"取得の失敗で終了した状態を変更しない", steamdomain.CollectionOptionsConfig{}, steamdomain.ReviewPage{Reviews: stateReviews(1), NextCursor: "A"}, steamdomain.StopFailed, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := steamdomain.NewCollectionState(440, collectionOptions(t, tc.options))
			if err != nil {
				t.Fatal(err)
			}
			err = state.AddPage(tc.page)
			if (err != nil) != tc.cycle || (tc.cycle && !errors.Is(err, steamdomain.ErrCursorCycle)) {
				t.Fatal(err)
			}
			if tc.fail {
				state.Fail()
			}
			before := state.Result()
			if before.StopReason != tc.reason {
				t.Fatalf("初期の終了理由=%s", before.StopReason)
			}
			state.Fail()
			if err := state.AddPage(steamdomain.ReviewPage{Reviews: stateReviews(9), NextCursor: "Z"}); err != nil {
				t.Fatal(err)
			}
			after := state.Result()
			if !reflect.DeepEqual(after, before) {
				t.Fatalf("終了後に変更した: before=%#v after=%#v", before, after)
			}
			if next, ok := state.NextPage(); ok || next != (steamdomain.PageOptions{}) {
				t.Fatalf("終了後の next=%#v available=%v", next, ok)
			}
		})
	}
}
