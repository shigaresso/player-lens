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

func TestCollectReviewsResponseLifetime(t *testing.T) {
	fetchFailure := errors.New("fetch failed")
	responseFailure := errors.New("response failed")
	for _, tc := range []struct {
		name        string
		fetchErr    error
		responseErr error
	}{
		{"成功結果の処理完了まで枠を保持する", nil, nil},
		{"結果処理の失敗後も枠とcontextを解放する", nil, responseFailure},
		{"取得失敗の結果処理中も枠を保持する", fetchFailure, nil},
		{"取得と結果処理の両方が失敗しても枠を返す", fetchFailure, responseFailure},
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
				return steamdomain.ReviewPage{}, tc.fetchErr
			})
			responses := 0
			err = mcpapplication.CollectReviews(t.Context(), fetcher, 440, steamdomain.CollectionOptions{}, gate, func(result mcpapplication.Result) error {
				responses++
				if !errors.Is(result.Err, tc.fetchErr) || result.Rejection != mcpdomain.NotRejected || result.Collection.AppID != 440 {
					t.Errorf("result=%+v", result)
				}
				release, reason := gate.Acquire()
				if release != nil {
					release()
				}
				if reason != mcpdomain.ConcurrentLimitReached {
					t.Errorf("結果処理中の同時枠=%v", reason)
				}
				return tc.responseErr
			})
			if err != tc.responseErr || responses != 1 {
				t.Fatalf("err=%v responses=%d", err, responses)
			}
			if fetchedContext == nil || !errors.Is(fetchedContext.Err(), context.Canceled) || t.Context().Err() != nil {
				t.Fatal("所有する context だけを解放しなかった")
			}
			release, reason := gate.Acquire()
			if reason != mcpdomain.NotRejected {
				t.Fatalf("終了後の同時枠=%v", reason)
			}
			release()
			if release, reason := gate.Acquire(); release != nil || reason != mcpdomain.MinuteLimitReached {
				t.Fatalf("開始回数を戻した: %v", reason)
			}
		})
	}
}

func TestCollectReviewsResponseFailureWithoutAcquisition(t *testing.T) {
	for _, tc := range []struct {
		name      string
		cancelled bool
	}{
		{"受付拒否の結果処理が失敗しても他の処理の枠を返さない", false},
		{"開始前キャンセルの結果処理が失敗しても枠を消費しない", true},
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
			failure := errors.New("response failed")
			responses := 0
			err = mcpapplication.CollectReviews(ctx, fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				t.Fatal("受付していない収集で取得した")
				return steamdomain.ReviewPage{}, nil
			}), 440, steamdomain.CollectionOptions{}, gate, func(result mcpapplication.Result) error {
				responses++
				if tc.cancelled {
					if !errors.Is(result.Err, context.Canceled) || result.Rejection != mcpdomain.NotRejected {
						t.Errorf("cancelled result=%+v", result)
					}
				} else if result.Err != nil || result.Rejection != mcpdomain.ConcurrentLimitReached {
					t.Errorf("rejected result=%+v", result)
				}
				return failure
			})
			if err != failure || responses != 1 {
				t.Fatalf("err=%v responses=%d", err, responses)
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
