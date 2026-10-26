package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectionReleasesSlotAfterFailure(t *testing.T) {
	for _, tc := range []struct {
		name string
		code string
	}{
		{"途中失敗でも枠を返却し開始回数は戻さない", "fetch_failed"},
		{"実行中のキャンセルでも枠を返却し開始回数は戻さない", "cancelled"},
		{"取得期限でも枠を返却し開始回数は戻さない", "deadline_exceeded"},
		{"応答量超過でも枠を返却し開始回数は戻さない", "response_too_large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gate := mcpdomain.NewLimiter(testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 2, RequestsPerDay: 10, MaxConcurrent: 1}), time.Now)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			first := true
			calls := 0
			fetcher := testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				calls++
				if !first {
					return steamdomain.ReviewPage{}, nil
				}

				switch tc.code {
				case "fetch_failed":
					if calls == 1 {
						return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 11}}, NextCursor: "next"}, nil
					}

					return steamdomain.ReviewPage{}, errors.New("fetch failed")
				case "cancelled":
					cancel()
					return steamdomain.ReviewPage{}, ctx.Err()
				case "deadline_exceeded":
					return steamdomain.ReviewPage{}, context.DeadlineExceeded
				default:
					return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 10, Text: strings.Repeat("x", 70000)}}, NextCursor: "next"}, nil
				}
			})
			count := 2
			if tc.code == "response_too_large" {
				count = 1
			}

			input, err := json.Marshal(map[string]any{"app_id": 440, "max_reviews": count})
			if err != nil {
				t.Fatal(err)
			}

			result, err := collect(ctx, fetcher, input, gate)
			if err != nil || !result.IsError || testutil.DecodeOutput(t, result).ErrorCode != tc.code {
				t.Fatalf("初回=%+v %v", result, err)
			}

			first = false
			result, err = collect(context.Background(), fetcher, input, gate)
			if err != nil || result.IsError {
				t.Fatalf("枠を返却しなかった: %+v %v", result, err)
			}

			before := calls
			result, err = collect(context.Background(), fetcher, input, gate)
			if err != nil || !result.IsError || testutil.DecodeOutput(t, result).ErrorCode != "rate_limited" || calls != before {
				t.Fatalf("開始回数を戻した: %+v %v", result, err)
			}
		})
	}
}

func TestCollectionBusyResponse(t *testing.T) {
	gate := mcpdomain.NewLimiter(testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 10, MaxConcurrent: 1}), time.Now)
	release, reason := gate.Acquire()
	if reason != mcpdomain.NotRejected {
		t.Fatal(reason)
	}

	defer release()

	result, err := collect(context.Background(), testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		t.Fatal("同時枠のない呼び出しで取得した")
		return steamdomain.ReviewPage{}, nil
	}), json.RawMessage(`{"app_id":440}`), gate)
	if err != nil {
		t.Fatal(err)
	}

	out := testutil.DecodeOutput(t, result)
	if !result.IsError || out.ErrorCode != "busy" || out.Collection.AppID != 440 || out.Collection.StopReason != steamdomain.StopFailed || out.Collection.PagesFetched != 0 || out.ReturnedCount != 0 || len(out.Collection.Reviews) != 0 {
		t.Fatal(out)
	}
}
