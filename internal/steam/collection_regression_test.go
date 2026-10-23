package steam_test

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strconv"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestCollectReviewsInvalidInput(t *testing.T) {
	for _, tt := range []struct {
		name       string
		appID      uint32
		options    steam.CollectionOptions
		nilFetcher bool
	}{
		{"AppIDゼロを拒否する", 0, steam.CollectionOptions{}, false},
		{"取得依存なしを拒否する", 440, steam.CollectionOptions{}, true},
		{"不正な言語を拒否する", 440, steam.CollectionOptions{Language: "Japanese"}, false},
		{"ページ件数の上限超過を拒否する", 440, steam.CollectionOptions{PageSize: 101}, false},
		{"収集件数の上限超過を拒否する", 440, steam.CollectionOptions{MaxReviews: 10001}, false},
		{"ページ上限の超過を拒否する", 440, steam.CollectionOptions{MaxPages: 101}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var fetcher steam.PageFetcher = pageFetcherFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
				t.Fatal("不正入力で取得した")
				return steam.ReviewPage{}, nil
			})
			if tt.nilFetcher {
				fetcher = nil
			}
			got, err := steam.CollectReviews(context.Background(), fetcher, tt.appID, tt.options)
			if !errors.Is(err, steam.ErrInvalidArgument) || !reflect.DeepEqual(got, steam.Collection{}) {
				t.Fatalf("result=%#v error=%v", got, err)
			}
		})
	}
}

func TestCollectReviewsMaximumReviews(t *testing.T) {
	calls := 0
	fetcher := pageFetcherFunc(func(_ context.Context, appID uint32, options steam.PageOptions) (steam.ReviewPage, error) {
		calls++
		if calls > 100 || options.PageSize != 100 {
			t.Fatalf("calls=%d options=%#v", calls, options)
		}
		reviews := make([]steam.Review, 100)
		for i := range reviews {
			reviews[i].RecommendationID = steam.RecommendationID((calls-1)*100 + i + 1)
		}
		return steam.ReviewPage{AppID: appID, Options: options, Reviews: reviews, NextCursor: strconv.Itoa(calls)}, nil
	})
	got, err := steam.CollectReviews(context.Background(), fetcher, 440, steam.CollectionOptions{MaxReviews: 10000})
	if err != nil || calls != 100 || got.PagesFetched != 100 || len(got.Reviews) != 10000 || got.StopReason != steam.StopReviewLimit || got.Reviews[0].RecommendationID != 1 || got.Reviews[9999].RecommendationID != 10000 {
		t.Fatalf("reviews=%d pages=%d calls=%d reason=%s error=%v", len(got.Reviews), got.PagesFetched, calls, got.StopReason, err)
	}
}

func TestCollectReviewsFailureReleasesContext(t *testing.T) {
	for _, tt := range []struct {
		name  string
		cause error
	}{
		{"取得失敗でも解放する", io.ErrUnexpectedEOF},
		{"循環でも解放する", steam.ErrCursorCycle},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var captured context.Context
			fetcher := pageFetcherFunc(func(ctx context.Context, appID uint32, options steam.PageOptions) (steam.ReviewPage, error) {
				captured = ctx
				if tt.cause == steam.ErrCursorCycle {
					return steam.ReviewPage{AppID: appID, Options: options, Reviews: collectionReviews(1), NextCursor: "*"}, nil
				}
				return steam.ReviewPage{}, tt.cause
			})
			got, err := steam.CollectReviews(context.Background(), fetcher, 440, steam.CollectionOptions{MaxReviews: 2})
			if !errors.Is(err, tt.cause) || got.StopReason != steam.StopFailed || captured == nil || !errors.Is(captured.Err(), context.Canceled) {
				t.Fatalf("result=%#v error=%v", got, err)
			}
		})
	}
}
