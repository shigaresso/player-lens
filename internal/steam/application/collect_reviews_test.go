package application_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type pageFetcherFunc func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error)

func (f pageFetcherFunc) FetchReviewsPage(ctx context.Context, appID steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
	return f(ctx, appID, options)
}

func collectionReviews(ids ...steamdomain.RecommendationID) []steamdomain.Review {
	result := make([]steamdomain.Review, 0, len(ids))
	for _, id := range ids {
		result = append(result, steamdomain.Review{RecommendationID: id, Text: fmt.Sprintf("本文%d", id), Language: "english"})
	}
	return result
}

func TestCollectReviewsPages(t *testing.T) {
	cases := []struct {
		name     string
		options  steamdomain.CollectionOptionsConfig
		pages    []steamdomain.ReviewPage
		requests []steamdomain.PageOptions
		want     []steamdomain.Review
		reason   steamdomain.StopReason
	}{
		{"空の先頭ページで終了する", steamdomain.CollectionOptionsConfig{}, []steamdomain.ReviewPage{{}}, []steamdomain.PageOptions{{Language: "english", Cursor: "*", PageSize: 100}}, []steamdomain.Review{}, steamdomain.StopExhausted},
		{"短いページでも次のカーソルを取得し空ページで終了する", steamdomain.CollectionOptionsConfig{PageSize: 2, MaxReviews: 5}, []steamdomain.ReviewPage{{Reviews: collectionReviews(1, 2), NextCursor: "A+/="}, {Reviews: collectionReviews(3), NextCursor: "B"}, {}}, []steamdomain.PageOptions{{Language: "english", Cursor: "*", PageSize: 2}, {Language: "english", Cursor: "A+/=", PageSize: 2}, {Language: "english", Cursor: "B", PageSize: 2}}, collectionReviews(1, 2, 3), steamdomain.StopExhausted},
		{"残件数を要求し件数上限をページ上限より優先する", steamdomain.CollectionOptionsConfig{PageSize: 2, MaxReviews: 3, MaxPages: 2}, []steamdomain.ReviewPage{{Reviews: collectionReviews(1, 2), NextCursor: "A"}, {Reviews: collectionReviews(3), NextCursor: "B"}}, []steamdomain.PageOptions{{Language: "english", Cursor: "*", PageSize: 2}, {Language: "english", Cursor: "A", PageSize: 1}}, collectionReviews(1, 2, 3), steamdomain.StopReviewLimit},
		{"重複は最初の本文と順序を維持する", steamdomain.CollectionOptionsConfig{PageSize: 3, MaxReviews: 4}, []steamdomain.ReviewPage{{Reviews: collectionReviews(1, 2), NextCursor: "A"}, {Reviews: []steamdomain.Review{{RecommendationID: 2, Text: "更新された本文"}, {RecommendationID: 3, Text: "本文3", Language: "english"}}, NextCursor: "B"}, {Reviews: collectionReviews(4), NextCursor: "C"}}, []steamdomain.PageOptions{{Language: "english", Cursor: "*", PageSize: 3}, {Language: "english", Cursor: "A", PageSize: 2}, {Language: "english", Cursor: "B", PageSize: 1}}, collectionReviews(1, 2, 3, 4), steamdomain.StopReviewLimit},
		{"ページ上限で追加取得を止める", steamdomain.CollectionOptionsConfig{PageSize: 2, MaxReviews: 10, MaxPages: 2}, []steamdomain.ReviewPage{{Reviews: collectionReviews(1, 2), NextCursor: "A"}, {Reviews: collectionReviews(3, 4), NextCursor: "B"}}, []steamdomain.PageOptions{{Language: "english", Cursor: "*", PageSize: 2}, {Language: "english", Cursor: "A", PageSize: 2}}, collectionReviews(1, 2, 3, 4), steamdomain.StopPageLimit},
		{"重複だけでもページ上限で終了する", steamdomain.CollectionOptionsConfig{PageSize: 2, MaxReviews: 2, MaxPages: 2}, []steamdomain.ReviewPage{{Reviews: collectionReviews(1), NextCursor: "A"}, {Reviews: collectionReviews(1), NextCursor: "B"}}, []steamdomain.PageOptions{{Language: "english", Cursor: "*", PageSize: 2}, {Language: "english", Cursor: "A", PageSize: 1}}, collectionReviews(1), steamdomain.StopPageLimit},
		{"空ページをページ上限より優先する", steamdomain.CollectionOptionsConfig{Language: "japanese", PageSize: 1, MaxReviews: 2, MaxPages: 2}, []steamdomain.ReviewPage{{Reviews: collectionReviews(1), NextCursor: "A"}, {}}, []steamdomain.PageOptions{{Language: "japanese", Cursor: "*", PageSize: 1}, {Language: "japanese", Cursor: "A", PageSize: 1}}, collectionReviews(1), steamdomain.StopExhausted},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for i := range tt.requests {
				tt.requests[i].ReviewType = steamdomain.ReviewTypeAll
			}
			var requests []steamdomain.PageOptions
			fetcher := pageFetcherFunc(func(_ context.Context, appID steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				if appID != 440 || len(requests) >= len(tt.pages) {
					t.Fatalf("予期しない取得: appID=%d 回数=%d", appID, len(requests)+1)
				}
				page := tt.pages[len(requests)]
				requests = append(requests, options)
				page.AppID, page.Options = appID, options
				return page, nil
			})
			got, err := steamapplication.CollectReviews(context.Background(), fetcher, 440, tt.options)
			wantOptions, normalizeErr := steamdomain.NormalizeCollectionOptions(tt.options)
			if err != nil || normalizeErr != nil || got.AppID != 440 || got.Options != wantOptions || got.PagesFetched != len(tt.pages) || got.StopReason != tt.reason || !reflect.DeepEqual(got.Reviews, tt.want) || !reflect.DeepEqual(requests, tt.requests) {
				t.Fatalf("result=%#v requests=%#v error=%v", got, requests, err)
			}
		})
	}
}
