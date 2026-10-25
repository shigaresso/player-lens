package domain_test

import (
	"reflect"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectionStateFail(t *testing.T) {
	for _, tc := range []struct {
		name  string
		pages []steamdomain.ReviewPage
		want  []steamdomain.Review
	}{
		{"先頭の失敗では空の部分結果を保持する", nil, []steamdomain.Review{}},
		{"途中の失敗では成功したページだけを保持する", []steamdomain.ReviewPage{{Reviews: stateReviews(1), NextCursor: "A"}}, stateReviews(1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := steamdomain.CollectionOptionsConfig{Language: "japanese", PageSize: 2, MaxReviews: 3, MaxPages: 4}
			state, err := steamdomain.NewCollectionState(440, collectionOptions(t, options))
			if err != nil {
				t.Fatal(err)
			}

			for _, page := range tc.pages {
				if err := state.AddPage(page); err != nil {
					t.Fatal(err)
				}
			}

			state.Fail()
			result := state.Result()
			if result.AppID != 440 || result.Options != options || result.PagesFetched != len(tc.pages) || result.StopReason != steamdomain.StopFailed || !reflect.DeepEqual(result.Reviews, tc.want) {
				t.Fatalf("result=%#v", result)
			}

			if next, ok := state.NextPage(); ok || next != (steamdomain.PageOptions{}) {
				t.Fatalf("失敗後の next=%#v available=%v", next, ok)
			}
		})
	}
}
