package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type collectionContextKey struct{}

func TestCollectReviewsContextDeadline(t *testing.T) {
	for _, tt := range []struct {
		name     string
		duration time.Duration
	}{
		{"期限なしでも最大2分にする", 0},
		{"長い期限は最大2分にする", 10 * time.Minute},
		{"呼び出し側の短い期限を保持する", 10 * time.Second},
	} {
		t.Run(tt.name, func(t *testing.T) {
			parent := context.WithValue(context.Background(), collectionContextKey{}, "trace")
			parent, cancel := context.WithCancel(parent)
			if tt.duration != 0 {
				cancel()
				parent, cancel = context.WithTimeout(context.WithValue(context.Background(), collectionContextKey{}, "trace"), tt.duration)
			}
			defer cancel()
			start := time.Now()
			var captured context.Context
			fetcher := pageFetcherFunc(func(ctx context.Context, appID steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				captured = ctx
				deadline, ok := ctx.Deadline()
				if !ok {
					t.Error("収集の期限がない")
				} else if tt.duration == 10*time.Second {
					parentDeadline, _ := parent.Deadline()
					if !deadline.Equal(parentDeadline) {
						t.Errorf("短い期限を変更した: %v", deadline)
					}
				} else if remaining := deadline.Sub(start); remaining < 119*time.Second || remaining > 121*time.Second {
					t.Errorf("収集の期限=%v", remaining)
				}
				if ctx.Value(collectionContextKey{}) != "trace" {
					t.Error("呼び出し側の context 値を失った")
				}
				return steamdomain.ReviewPage{AppID: appID, Options: options, Reviews: collectionReviews(1), NextCursor: "A"}, nil
			})
			got, err := steamapplication.CollectReviews(parent, fetcher, 440, steamdomain.CollectionOptions{MaxReviews: 1})
			if err != nil || got.StopReason != steamdomain.StopReviewLimit {
				t.Fatalf("result=%#v error=%v", got, err)
			}
			if captured == nil || !errors.Is(captured.Err(), context.Canceled) {
				t.Error("収集後に派生 context を解放していない")
			}
			if parent.Err() != nil {
				t.Errorf("呼び出し側の context を終了した: %v", parent.Err())
			}
		})
	}
}

func TestCollectReviewsStoppedContext(t *testing.T) {
	for _, tt := range []struct {
		name    string
		expired bool
		cause   error
	}{
		{"キャンセル済みは取得しない", false, context.Canceled},
		{"期限切れは取得しない", true, context.DeadlineExceeded},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if tt.expired {
				ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
				defer cancel()
			}
			fetcher := pageFetcherFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				t.Fatal("終了済み context で取得した")
				return steamdomain.ReviewPage{}, nil
			})
			got, err := steamapplication.CollectReviews(ctx, fetcher, 440, steamdomain.CollectionOptions{})
			if !errors.Is(err, tt.cause) || got.AppID != 440 || got.StopReason != steamdomain.StopFailed || got.PagesFetched != 0 || !reflect.DeepEqual(got.Reviews, []steamdomain.Review{}) {
				t.Fatalf("result=%#v error=%v", got, err)
			}
		})
	}
}

func TestCollectReviewsCancellationBetweenPages(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	var captured context.Context
	fetcher := pageFetcherFunc(func(pageCtx context.Context, appID steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		calls++
		if calls > 1 {
			t.Fatal("キャンセル後に次のページを取得した")
		}
		captured = pageCtx
		cancel()
		return steamdomain.ReviewPage{AppID: appID, Options: options, Reviews: collectionReviews(1), NextCursor: "A"}, nil
	})
	got, err := steamapplication.CollectReviews(ctx, fetcher, 440, steamdomain.CollectionOptions{PageSize: 1, MaxReviews: 2})
	if !errors.Is(err, context.Canceled) || got.StopReason != steamdomain.StopFailed || got.PagesFetched != 1 || calls != 1 || !reflect.DeepEqual(got.Reviews, collectionReviews(1)) || captured == nil || !errors.Is(captured.Err(), context.Canceled) {
		t.Fatalf("result=%#v calls=%d error=%v", got, calls, err)
	}
}
