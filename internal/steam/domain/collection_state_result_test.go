package domain_test

import (
	"reflect"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectionStateResultOwnership(t *testing.T) {
	for _, tc := range []struct {
		name    string
		options steamdomain.CollectionOptionsConfig
		fail    bool
		reason  steamdomain.StopReason
	}{
		{"進行中の結果を書き換えても状態は変わらない", steamdomain.CollectionOptionsConfig{}, false, ""},
		{"成功結果を書き換えても状態は変わらない", steamdomain.CollectionOptionsConfig{MaxReviews: 1}, false, steamdomain.StopReviewLimit},
		{"失敗結果を書き換えても状態は変わらない", steamdomain.CollectionOptionsConfig{}, true, steamdomain.StopFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := steamdomain.NewCollectionState(440, tc.options)
			if err != nil {
				t.Fatal(err)
			}
			first := steamdomain.Review{RecommendationID: 1, Text: "最初の本文", TimestampCreated: 10, TimestampUpdated: 20}
			if err := state.AddPage(steamdomain.ReviewPage{Reviews: []steamdomain.Review{first}, NextCursor: "A"}); err != nil {
				t.Fatal(err)
			}
			if tc.fail {
				state.Fail()
			}
			returned := state.Result()
			returned.AppID = 570
			returned.Options.Language = "japanese"
			returned.Reviews[0].RecommendationID = 9
			returned.Reviews[0].Text = "外部で変更した本文"
			normalized, err := steamdomain.NormalizeCollectionOptions(tc.options)
			if err != nil {
				t.Fatal(err)
			}
			want := steamdomain.Collection{AppID: 440, Options: normalized, Reviews: []steamdomain.Review{first}, PagesFetched: 1, StopReason: tc.reason}
			if got := state.Result(); !reflect.DeepEqual(got, want) {
				t.Fatalf("result=%#v; want=%#v", got, want)
			}
		})
	}
}
