package application_test

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strconv"
	"testing"

	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectReviewsInvalidInput(t *testing.T) {
	for _, tt := range []struct {
		name       string
		appID      steamdomain.AppID
		options    steamdomain.CollectionOptions
		nilFetcher bool
	}{
		{"AppIDゼロを拒否する", 0, steamdomain.CollectionOptions{}, false},
		{"取得依存なしを拒否する", 440, steamdomain.CollectionOptions{}, true},
		{"不正な言語を拒否する", 440, steamdomain.CollectionOptions{Language: "Japanese"}, false},
		{"ページ件数の上限超過を拒否する", 440, steamdomain.CollectionOptions{PageSize: 101}, false},
		{"収集件数の上限超過を拒否する", 440, steamdomain.CollectionOptions{MaxReviews: 10001}, false},
		{"ページ上限の超過を拒否する", 440, steamdomain.CollectionOptions{MaxPages: 101}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var fetcher steamapplication.PageFetcher = pageFetcherFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				t.Fatal("不正入力で取得した")
				return steamdomain.ReviewPage{}, nil
			})
			if tt.nilFetcher {
				fetcher = nil
			}
			got, err := steamapplication.CollectReviews(context.Background(), fetcher, tt.appID, tt.options)
			if !errors.Is(err, steamdomain.ErrInvalidArgument) || !reflect.DeepEqual(got, steamdomain.Collection{}) {
				t.Fatalf("result=%#v error=%v", got, err)
			}
		})
	}
}

func TestCollectReviewsMaximumReviews(t *testing.T) {
	calls := 0
	fetcher := pageFetcherFunc(func(_ context.Context, appID steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		calls++
		if calls > 100 || options.PageSize != 100 {
			t.Fatalf("calls=%d options=%#v", calls, options)
		}
		reviews := make([]steamdomain.Review, 100)
		for i := range reviews {
			reviews[i].RecommendationID = steamdomain.RecommendationID((calls-1)*100 + i + 1)
		}
		return steamdomain.ReviewPage{AppID: appID, Options: options, Reviews: reviews, NextCursor: strconv.Itoa(calls)}, nil
	})
	got, err := steamapplication.CollectReviews(context.Background(), fetcher, 440, steamdomain.CollectionOptions{MaxReviews: 10000})
	if err != nil || calls != 100 || got.PagesFetched != 100 || len(got.Reviews) != 10000 || got.StopReason != steamdomain.StopReviewLimit || got.Reviews[0].RecommendationID != 1 || got.Reviews[9999].RecommendationID != 10000 {
		t.Fatalf("reviews=%d pages=%d calls=%d reason=%s error=%v", len(got.Reviews), got.PagesFetched, calls, got.StopReason, err)
	}
}

func TestCollectReviewsFailureReleasesContext(t *testing.T) {
	for _, tt := range []struct {
		name  string
		cause error
	}{
		{"取得失敗でも解放する", io.ErrUnexpectedEOF},
		{"循環でも解放する", steamapplication.ErrCursorCycle},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var captured context.Context
			fetcher := pageFetcherFunc(func(ctx context.Context, appID steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				captured = ctx
				if tt.cause == steamapplication.ErrCursorCycle {
					return steamdomain.ReviewPage{AppID: appID, Options: options, Reviews: collectionReviews(1), NextCursor: "*"}, nil
				}
				return steamdomain.ReviewPage{}, tt.cause
			})
			got, err := steamapplication.CollectReviews(context.Background(), fetcher, 440, steamdomain.CollectionOptions{MaxReviews: 2})
			if !errors.Is(err, tt.cause) || got.StopReason != steamdomain.StopFailed || captured == nil || !errors.Is(captured.Err(), context.Canceled) {
				t.Fatalf("result=%#v error=%v", got, err)
			}
		})
	}
}
