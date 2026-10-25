package domain_test

import (
	"reflect"
	"testing"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func stateReviews(ids ...steamdomain.RecommendationID) []steamdomain.Review {
	reviews := make([]steamdomain.Review, 0, len(ids))
	for _, id := range ids {
		reviews = append(reviews, steamdomain.Review{RecommendationID: id})
	}
	return reviews
}

func TestCollectionStatePages(t *testing.T) {
	first := steamdomain.Review{RecommendationID: 0, Text: "最初の本文", Language: "japanese", Recommended: true, TimestampCreated: 10, TimestampUpdated: 20}
	for _, tc := range []struct {
		name    string
		options steamdomain.CollectionOptionsConfig
		pages   []steamdomain.ReviewPage
		want    []steamdomain.Review
		reason  steamdomain.StopReason
		next    steamdomain.PageOptions
	}{
		{"短いページを反映して残件数と加工しないカーソルを返す", steamdomain.CollectionOptionsConfig{PageSize: 3, MaxReviews: 3, MaxPages: 3}, []steamdomain.ReviewPage{{Reviews: stateReviews(1), NextCursor: "A+/="}}, stateReviews(1), "", steamdomain.PageOptions{Language: "english", Cursor: "A+/=", PageSize: 2}},
		{"重複だけのページでも継続する", steamdomain.CollectionOptionsConfig{PageSize: 2, MaxReviews: 3, MaxPages: 3}, []steamdomain.ReviewPage{{Reviews: stateReviews(1), NextCursor: "A"}, {Reviews: stateReviews(1), NextCursor: "B"}}, stateReviews(1), "", steamdomain.PageOptions{Language: "english", Cursor: "B", PageSize: 2}},
		{"数値 ID が同じなら最初の本文と順序を保持する", steamdomain.CollectionOptionsConfig{MaxReviews: 2}, []steamdomain.ReviewPage{{Reviews: []steamdomain.Review{first}, NextCursor: "A"}, {Reviews: []steamdomain.Review{{RecommendationID: 0, Text: "更新された本文"}, {RecommendationID: 1}}, NextCursor: "B"}}, []steamdomain.Review{first, {RecommendationID: 1}}, steamdomain.StopReviewLimit, steamdomain.PageOptions{}},
		{"空ページも成功数へ含め走査終了をページ上限より優先する", steamdomain.CollectionOptionsConfig{MaxPages: 1}, []steamdomain.ReviewPage{{}}, []steamdomain.Review{}, steamdomain.StopExhausted, steamdomain.PageOptions{}},
		{"件数上限をページ上限より優先する", steamdomain.CollectionOptionsConfig{MaxReviews: 1, MaxPages: 1}, []steamdomain.ReviewPage{{Reviews: stateReviews(1), NextCursor: "A"}}, stateReviews(1), steamdomain.StopReviewLimit, steamdomain.PageOptions{}},
		{"重複除外後の件数が足りなくてもページ上限で終了する", steamdomain.CollectionOptionsConfig{MaxReviews: 5, MaxPages: 2}, []steamdomain.ReviewPage{{Reviews: stateReviews(1), NextCursor: "A"}, {Reviews: stateReviews(1), NextCursor: "B"}}, stateReviews(1), steamdomain.StopPageLimit, steamdomain.PageOptions{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state, err := steamdomain.NewCollectionState(440, tc.options)
			if err != nil {
				t.Fatal(err)
			}
			for _, page := range tc.pages {
				if _, ok := state.NextPage(); !ok {
					t.Fatal("終了条件の前に停止した")
				}
				if err := state.AddPage(page); err != nil {
					t.Fatal(err)
				}
			}
			result := state.Result()
			normalized, err := steamdomain.NormalizeCollectionOptions(tc.options)
			if err != nil {
				t.Fatal(err)
			}
			if result.AppID != 440 || result.Options != normalized || result.PagesFetched != len(tc.pages) || result.StopReason != tc.reason || !reflect.DeepEqual(result.Reviews, tc.want) {
				t.Fatalf("result=%#v", result)
			}
			next, ok := state.NextPage()
			if ok != (tc.reason == "") || next != tc.next {
				t.Fatalf("next=%#v available=%v", next, ok)
			}
		})
	}
}
