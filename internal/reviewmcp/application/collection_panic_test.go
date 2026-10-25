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

func TestCollectReviewsPanicReleasesSlot(t *testing.T) {
	for _, tc := range []struct {
		name          string
		duringFetch   bool
		wantResponses int
	}{
		{"収集中のpanicでも枠を返して原因を伝える", true, 0},
		{"応答変換中のpanicでも枠を返して原因を伝える", false, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits, err := mcpdomain.NewLimits(mcpdomain.LimitsConfig{RequestsPerMinute: 2, RequestsPerDay: 10, MaxConcurrent: 1})
			if err != nil {
				t.Fatal(err)
			}
			gate := mcpdomain.NewLimiter(limits, time.Now)
			cause := errors.New("panic cause")
			var fetchedContext context.Context
			fetcher := fetchFunc(func(ctx context.Context, _ steamdomain.AppID, _ steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				fetchedContext = ctx
				if tc.duringFetch {
					panic(cause)
				}
				return steamdomain.ReviewPage{}, nil
			})
			responses := 0
			func() {
				defer func() {
					if got := recover(); got != cause {
						t.Errorf("panic の原因=%v; want=%v", got, cause)
					}
				}()
				err := mcpapplication.CollectReviews(t.Context(), fetcher, 440, steamdomain.CollectionOptions{}, gate, func(mcpapplication.Result) error {
					responses++
					panic(cause)
				})
				t.Errorf("panic を呼び出し側へ伝えなかった: err=%v", err)
			}()
			if responses != tc.wantResponses {
				t.Fatalf("応答変換の呼び出し回数=%d; want=%d", responses, tc.wantResponses)
			}
			if fetchedContext == nil || !errors.Is(fetchedContext.Err(), context.Canceled) || t.Context().Err() != nil {
				t.Fatal("panic 後に所有する context だけを解放しなかった")
			}
			release, reason := gate.Acquire()
			if reason != mcpdomain.NotRejected {
				t.Fatalf("panic 後に同時枠を返却しなかった: %v", reason)
			}
			release()
			if release, reason := gate.Acquire(); release != nil || reason != mcpdomain.MinuteLimitReached {
				t.Fatalf("panic 後に開始回数を戻した: %v", reason)
			}
		})
	}
}
