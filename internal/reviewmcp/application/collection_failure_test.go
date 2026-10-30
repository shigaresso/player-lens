package application_test

import (
	"context"
	"errors"
	"reflect"
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

			options := steamdomain.CollectionOptionsConfig{Language: "english", PageSize: 100, MaxReviews: 20, MaxPages: 3}
			result := mcpapplication.CollectReviews(t.Context(), fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				t.Fatal("拒否した操作で取得した")
				return steamdomain.ReviewPage{}, nil
			}), 440, collectionOptions(t, options), gate)
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
	}), 440, collectionOptions(t, steamdomain.CollectionOptionsConfig{}), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
	if !errors.Is(result.Err, cause) || result.Rejection != mcpdomain.NotRejected || result.Collection.StopReason != steamdomain.StopFailed || result.Collection.PagesFetched != 1 || len(result.Collection.Reviews) != 1 || result.Collection.Reviews[0].RecommendationID != 11 {
		t.Fatalf("result=%+v", result)
	}

	if fetchedContext == nil || fetchedContext.Err() != context.Canceled || t.Context().Err() != nil {
		t.Fatal("取得用 context の終了が正しくない")
	}
}

func TestCollectReviewsCancelledBeforeStart(t *testing.T) {
	for _, tc := range []struct {
		name    string
		expired bool
		cause   error
	}{
		{"開始前キャンセルでは取得せず失敗結果を返す", false, context.Canceled},
		{"開始前期限超過では取得せず失敗結果を返す", true, context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits, err := mcpdomain.NewLimits(mcpdomain.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 1, MaxConcurrent: 1})
			if err != nil {
				t.Fatal(err)
			}

			clockCalls := 0
			gate := mcpdomain.NewLimiter(limits, func() time.Time {
				clockCalls++
				return time.Now()
			})
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			if tc.expired {
				ctx, cancel = context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
			}

			defer cancel()

			config := steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewTypeNegative, Language: "japanese", PageSize: 10, MaxReviews: 20, MaxPages: 3}
			fetchCalls := 0
			result := mcpapplication.CollectReviews(ctx, fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				fetchCalls++
				return steamdomain.ReviewPage{}, nil
			}), 440, collectionOptions(t, config), gate)
			want := steamdomain.Collection{AppID: 440, Options: config, Reviews: []steamdomain.Review{}, StopReason: steamdomain.StopFailed}
			if !errors.Is(result.Err, tc.cause) || result.Rejection != mcpdomain.NotRejected || !reflect.DeepEqual(result.Collection, want) {
				t.Fatalf("result=%+v; want=%+v cause=%v", result, want, tc.cause)
			}

			if fetchCalls != 0 || clockCalls != 0 || t.Context().Err() != nil {
				t.Fatalf("取得=%d 時計=%d 親=%v", fetchCalls, clockCalls, t.Context().Err())
			}

			release, reason := gate.Acquire()
			if reason != mcpdomain.NotRejected {
				t.Fatalf("開始前の終了が利用枠を消費した: %v", reason)
			}

			defer release()
		})
	}
}
