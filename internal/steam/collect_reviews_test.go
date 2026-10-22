package steam_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

type pageFetcherFunc func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error)

func (f pageFetcherFunc) FetchReviewsPage(ctx context.Context, appID uint32, options steam.PageOptions) (steam.ReviewPage, error) {
	return f(ctx, appID, options)
}

func collectionReviews(ids ...string) []steam.Review {
	result := make([]steam.Review, 0, len(ids))
	for _, id := range ids {
		result = append(result, steam.Review{ID: id, Text: "本文" + id, Language: "english"})
	}
	return result
}

func TestCollectReviewsPages(t *testing.T) {
	cases := []struct {
		name     string
		options  steam.CollectionOptions
		pages    []steam.ReviewPage
		requests []steam.PageOptions
		want     []steam.Review
		reason   steam.StopReason
	}{
		{"空の先頭ページで終了する", steam.CollectionOptions{}, []steam.ReviewPage{{}}, []steam.PageOptions{{Language: "english", Cursor: "*", PageSize: 100}}, []steam.Review{}, steam.StopExhausted},
		{"短いページでも次のカーソルを取得し空ページで終了する", steam.CollectionOptions{PageSize: 2, MaxReviews: 5}, []steam.ReviewPage{{Reviews: collectionReviews("1", "2"), NextCursor: "A+/="}, {Reviews: collectionReviews("3"), NextCursor: "B"}, {}}, []steam.PageOptions{{Language: "english", Cursor: "*", PageSize: 2}, {Language: "english", Cursor: "A+/=", PageSize: 2}, {Language: "english", Cursor: "B", PageSize: 2}}, collectionReviews("1", "2", "3"), steam.StopExhausted},
		{"残件数を要求し件数上限をページ上限より優先する", steam.CollectionOptions{PageSize: 2, MaxReviews: 3, MaxPages: 2}, []steam.ReviewPage{{Reviews: collectionReviews("1", "2"), NextCursor: "A"}, {Reviews: collectionReviews("3"), NextCursor: "B"}}, []steam.PageOptions{{Language: "english", Cursor: "*", PageSize: 2}, {Language: "english", Cursor: "A", PageSize: 1}}, collectionReviews("1", "2", "3"), steam.StopReviewLimit},
		{"重複は最初の本文と順序を維持する", steam.CollectionOptions{PageSize: 3, MaxReviews: 4}, []steam.ReviewPage{{Reviews: collectionReviews("1", "2"), NextCursor: "A"}, {Reviews: []steam.Review{{ID: "2", Text: "更新された本文"}, {ID: "3", Text: "本文3", Language: "english"}}, NextCursor: "B"}, {Reviews: collectionReviews("4"), NextCursor: "C"}}, []steam.PageOptions{{Language: "english", Cursor: "*", PageSize: 3}, {Language: "english", Cursor: "A", PageSize: 2}, {Language: "english", Cursor: "B", PageSize: 1}}, collectionReviews("1", "2", "3", "4"), steam.StopReviewLimit},
		{"ページ上限で追加取得を止める", steam.CollectionOptions{PageSize: 2, MaxReviews: 10, MaxPages: 2}, []steam.ReviewPage{{Reviews: collectionReviews("1", "2"), NextCursor: "A"}, {Reviews: collectionReviews("3", "4"), NextCursor: "B"}}, []steam.PageOptions{{Language: "english", Cursor: "*", PageSize: 2}, {Language: "english", Cursor: "A", PageSize: 2}}, collectionReviews("1", "2", "3", "4"), steam.StopPageLimit},
		{"重複だけでもページ上限で終了する", steam.CollectionOptions{PageSize: 2, MaxReviews: 2, MaxPages: 2}, []steam.ReviewPage{{Reviews: collectionReviews("1"), NextCursor: "A"}, {Reviews: collectionReviews("1"), NextCursor: "B"}}, []steam.PageOptions{{Language: "english", Cursor: "*", PageSize: 2}, {Language: "english", Cursor: "A", PageSize: 1}}, collectionReviews("1"), steam.StopPageLimit},
		{"空ページをページ上限より優先する", steam.CollectionOptions{Language: "japanese", PageSize: 1, MaxReviews: 2, MaxPages: 2}, []steam.ReviewPage{{Reviews: collectionReviews("1"), NextCursor: "A"}, {}}, []steam.PageOptions{{Language: "japanese", Cursor: "*", PageSize: 1}, {Language: "japanese", Cursor: "A", PageSize: 1}}, collectionReviews("1"), steam.StopExhausted},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			for i := range tt.requests {
				tt.requests[i].ReviewType = steam.ReviewTypeAll
			}
			var requests []steam.PageOptions
			fetcher := pageFetcherFunc(func(_ context.Context, appID uint32, options steam.PageOptions) (steam.ReviewPage, error) {
				if appID != 440 || len(requests) >= len(tt.pages) {
					t.Fatalf("予期しない取得: appID=%d 回数=%d", appID, len(requests)+1)
				}
				page := tt.pages[len(requests)]
				requests = append(requests, options)
				page.AppID, page.Options = appID, options
				return page, nil
			})
			got, err := steam.CollectReviews(context.Background(), fetcher, 440, tt.options)
			wantOptions, normalizeErr := steam.NormalizeCollectionOptions(tt.options)
			if err != nil || normalizeErr != nil || got.AppID != 440 || got.Options != wantOptions || got.PagesFetched != len(tt.pages) || got.StopReason != tt.reason || !reflect.DeepEqual(got.Reviews, tt.want) || !reflect.DeepEqual(requests, tt.requests) {
				t.Fatalf("result=%#v requests=%#v error=%v", got, requests, err)
			}
		})
	}
}
