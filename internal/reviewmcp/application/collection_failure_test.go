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

func TestCollectReviewsRejection(t *testing.T) {
	for _, tc := range []struct {
		name     string
		config   mcpdomain.LimitsConfig
		occupied bool
		reason   mcpdomain.RejectionReason
	}{
		{"分次上限では取得せず条件と拒否理由を返す", mcpdomain.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 10, MaxConcurrent: 1}, false, mcpdomain.MinuteLimitReached},
		{"日次上限では取得せず条件と拒否理由を返す", mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 1, MaxConcurrent: 1}, false, mcpdomain.DailyLimitReached},
		{"同時枠がなければ取得せず条件と拒否理由を返す", mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 10, MaxConcurrent: 1}, true, mcpdomain.ConcurrentLimitReached},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits, err := mcpdomain.NewLimits(tc.config)
			if err != nil {
				t.Fatal(err)
			}
			gate := mcpdomain.NewLimiter(limits, time.Now)
			release, reason := gate.Acquire()
			if reason != mcpdomain.NotRejected {
				t.Fatal(reason)
			}
			t.Cleanup(release)
			if !tc.occupied {
				release()
			}
			options := steamdomain.CollectionOptions{Language: "english", PageSize: 100, MaxReviews: 20, MaxPages: 3}
			result := mcpapplication.CollectReviews(t.Context(), fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				t.Fatal("拒否した操作で取得した")
				return steamdomain.ReviewPage{}, nil
			}), 440, options, gate)
			if result.Err != nil || result.Rejection != tc.reason || result.Collection.AppID != 440 || result.Collection.Options != options || result.Collection.StopReason != steamdomain.StopFailed || result.Collection.PagesFetched != 0 || len(result.Collection.Reviews) != 0 {
				t.Fatalf("result=%+v", result)
			}
		})
	}
}

func TestCollectReviewsFailure(t *testing.T) {
	cause := errors.New("fetch failed")
	calls := 0
	var fetchedContext context.Context
	result := mcpapplication.CollectReviews(t.Context(), fetchFunc(func(ctx context.Context, _ steamdomain.AppID, _ steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		fetchedContext = ctx
		calls++
		if calls == 1 {
			return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 11}}, NextCursor: "next"}, nil
		}
		return steamdomain.ReviewPage{}, cause
	}), 440, steamdomain.CollectionOptions{}, mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
	if !errors.Is(result.Err, cause) || result.Rejection != mcpdomain.NotRejected || result.Collection.StopReason != steamdomain.StopFailed || result.Collection.PagesFetched != 1 || len(result.Collection.Reviews) != 1 || result.Collection.Reviews[0].RecommendationID != 11 {
		t.Fatalf("result=%+v", result)
	}
	if fetchedContext == nil || fetchedContext.Err() != context.Canceled || t.Context().Err() != nil {
		t.Fatal("取得用 context の終了が正しくない")
	}
}

func TestCollectReviewsCancelledBeforeStart(t *testing.T) {
	limits, err := mcpdomain.NewLimits(mcpdomain.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 1, MaxConcurrent: 1})
	if err != nil {
		t.Fatal(err)
	}
	gate := mcpdomain.NewLimiter(limits, time.Now)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	result := mcpapplication.CollectReviews(ctx, fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		t.Fatal("開始前キャンセルで取得した")
		return steamdomain.ReviewPage{}, nil
	}), 440, steamdomain.CollectionOptions{}, gate)
	if !errors.Is(result.Err, context.Canceled) || result.Rejection != mcpdomain.NotRejected || result.Collection.AppID != 440 || result.Collection.StopReason != steamdomain.StopFailed || result.Collection.PagesFetched != 0 {
		t.Fatalf("result=%+v", result)
	}
	release, reason := gate.Acquire()
	if reason != mcpdomain.NotRejected {
		t.Fatalf("キャンセル前の操作が利用枠を消費した: %v", reason)
	}
	release()
}
