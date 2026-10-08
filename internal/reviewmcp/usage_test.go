package reviewmcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestCollectionUsageBudget(t *testing.T) {
	for _, tc := range []struct {
		name   string
		limits Limits
		code   string
	}{
		{"分次の拒否は取得せず条件と空の結果を返す", Limits{RequestsPerMinute: 1, RequestsPerDay: 10, MaxConcurrent: 1}, "rate_limited"},
		{"日次の拒否は取得せず条件と空の結果を返す", Limits{RequestsPerMinute: 10, RequestsPerDay: 1, MaxConcurrent: 1}, "daily_limit_reached"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gate := newLimiter(tc.limits, time.Now)
			calls := 0
			fetcher := fetchFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
				calls++
				return steam.ReviewPage{}, nil
			})
			input := json.RawMessage(`{"app_id":440,"language":"all","max_reviews":1,"max_pages":1}`)
			result, err := collect(context.Background(), fetcher, input, gate)
			if err != nil || result.IsError {
				t.Fatalf("初回=%+v %v", result, err)
			}
			result, err = collect(context.Background(), fetcher, input, gate)
			if err != nil {
				t.Fatal(err)
			}
			out := decodeOutput(t, result)
			if !result.IsError || out.ErrorCode != tc.code || calls != 1 || out.ReturnedCount != 0 || out.Collection.PagesFetched != 0 || out.Collection.Reviews == nil || len(out.Collection.Reviews) != 0 || out.Collection.StopReason != steam.StopFailed || out.Collection.AppID != 440 || out.Collection.Options.Language != "all" || out.Collection.Options.MaxReviews != 1 || out.Collection.Options.MaxPages != 1 || out.Message == "" {
				t.Fatalf("拒否=%+v calls=%d", out, calls)
			}
		})
	}
	for _, tc := range []struct {
		name    string
		invalid bool
	}{
		{"不正入力は受付回数に含めない", true},
		{"開始前のキャンセルは受付回数に含めない", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gate := newLimiter(Limits{RequestsPerMinute: 1, RequestsPerDay: 1, MaxConcurrent: 1}, time.Now)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			input := json.RawMessage(`{"app_id":440}`)
			if tc.invalid {
				input = json.RawMessage(`{"app_id":0}`)
			} else {
				cancel()
			}
			fetcher := fetchFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
				return steam.ReviewPage{}, nil
			})
			result, err := collect(ctx, fetcher, input, gate)
			if err != nil || !result.IsError {
				t.Fatalf("拒否=%+v %v", result, err)
			}
			result, err = collect(context.Background(), fetcher, json.RawMessage(`{"app_id":440}`), gate)
			if err != nil || result.IsError {
				t.Fatalf("回数が消費された: %+v %v", result, err)
			}
		})
	}
}
