package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mcpapplication "github.com/shigaresso/player-lens/internal/reviewmcp/application"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectReviewsCollectionLifetime(t *testing.T) {
	for _, tc := range []struct {
		name     string
		fetchErr error
	}{
		{"収集中は枠を保持し成功結果を返す前に解放する", nil},
		{"収集中は枠を保持し失敗結果を返す前に解放する", errors.New("fetch failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits, err := mcpdomain.NewLimits(mcpdomain.LimitsConfig{RequestsPerMinute: 2, RequestsPerDay: 10, MaxConcurrent: 1})
			if err != nil {
				t.Fatal(err)
			}

			gate := mcpdomain.NewLimiter(limits, time.Now)
			var fetchedContext context.Context
			fetcher := fetchFunc(func(ctx context.Context, _ steamdomain.AppID, _ steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				fetchedContext = ctx
				release, reason := gate.Acquire()
				if release != nil {
					release()
				}

				if reason != mcpdomain.ConcurrentLimitReached {
					t.Errorf("収集中の同時枠=%v", reason)
				}

				return steamdomain.ReviewPage{}, tc.fetchErr
			})
			result := mcpapplication.CollectReviews(t.Context(), fetcher, 440, steamdomain.CollectionOptions{}, gate)
			if !errors.Is(result.Err, tc.fetchErr) || result.Rejection != mcpdomain.NotRejected || result.Collection.AppID != 440 {
				t.Fatalf("result=%+v", result)
			}

			if fetchedContext == nil || !errors.Is(fetchedContext.Err(), context.Canceled) || t.Context().Err() != nil {
				t.Fatal("所有する context だけを解放しなかった")
			}

			release, reason := gate.Acquire()
			if reason != mcpdomain.NotRejected {
				t.Fatalf("結果を受け取った時点の同時枠=%v", reason)
			}

			release()
			if release, reason := gate.Acquire(); release != nil || reason != mcpdomain.MinuteLimitReached {
				t.Fatalf("開始回数を戻した: %v", reason)
			}
		})
	}
}

func TestCollectReviewsWithoutAcquisition(t *testing.T) {
	for _, tc := range []struct {
		name      string
		cancelled bool
	}{
		{"受付拒否では他の処理の枠を返さない", false},
		{"開始前キャンセルでは枠を消費しない", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 10, MaxConcurrent: 1}
			if tc.cancelled {
				config.RequestsPerMinute = 1
				config.RequestsPerDay = 1
			}

			limits, err := mcpdomain.NewLimits(config)
			if err != nil {
				t.Fatal(err)
			}

			gate := mcpdomain.NewLimiter(limits, time.Now)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			var occupied func()
			if tc.cancelled {
				cancel()
			} else {
				occupied, _ = gate.Acquire()
				t.Cleanup(occupied)
			}

			result := mcpapplication.CollectReviews(ctx, fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				t.Fatal("受付していない収集で取得した")
				return steamdomain.ReviewPage{}, nil
			}), 440, steamdomain.CollectionOptions{}, gate)
			if tc.cancelled {
				if !errors.Is(result.Err, context.Canceled) || result.Rejection != mcpdomain.NotRejected {
					t.Fatalf("cancelled result=%+v", result)
				}
			} else if result.Err != nil || result.Rejection != mcpdomain.ConcurrentLimitReached {
				t.Fatalf("rejected result=%+v", result)
			}

			if occupied != nil {
				release, reason := gate.Acquire()
				if release != nil {
					release()
				}

				if reason != mcpdomain.ConcurrentLimitReached {
					t.Fatal("別の収集の同時枠を返した")
				}

				occupied()
			}

			release, reason := gate.Acquire()
			if reason != mcpdomain.NotRejected {
				t.Fatalf("未確保の同時枠を変化させた: %v", reason)
			}

			release()
		})
	}
}
