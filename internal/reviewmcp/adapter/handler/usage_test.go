package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectionUsageBudget(t *testing.T) {
	for _, tc := range []struct {
		name   string
		limits mcpdomain.Limits
		code   string
	}{
		{"分次の拒否は取得せず条件と空の結果を返す", testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 10, MaxConcurrent: 1}), "rate_limited"},
		{"日次の拒否は取得せず条件と空の結果を返す", testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 1, MaxConcurrent: 1}), "daily_limit_reached"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gate := mcpdomain.NewLimiter(tc.limits, time.Now)
			calls := 0
			fetcher := testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				calls++
				return steamdomain.ReviewPage{}, nil
			})
			input := json.RawMessage(`{"app_id":440,"language":"all","max_reviews":1,"max_pages":1}`)
			result, err := callCollectReviews(t, context.Background(), fetcher, input, gate)
			if err != nil || result.IsError {
				t.Fatalf("初回=%+v %v", result, err)
			}

			result, err = callCollectReviews(t, context.Background(), fetcher, input, gate)
			if err != nil {
				t.Fatal(err)
			}

			out := testutil.DecodeOutput(t, result)
			if !result.IsError || out.ErrorCode != tc.code || calls != 1 || out.ReturnedCount != 0 || out.Collection.PagesFetched != 0 || out.Collection.Reviews == nil || len(out.Collection.Reviews) != 0 || out.Collection.StopReason != steamdomain.StopFailed || out.Collection.AppID != 440 || out.Collection.Options.Language != "all" || out.Collection.Options.MaxReviews != 1 || out.Collection.Options.MaxPages != 1 || out.Message == "" {
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
			gate := mcpdomain.NewLimiter(testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 1, MaxConcurrent: 1}), time.Now)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			input := json.RawMessage(`{"app_id":440}`)
			if tc.invalid {
				input = json.RawMessage(`{"app_id":0}`)
			} else {
				cancel()
			}

			fetcher := testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				return steamdomain.ReviewPage{}, nil
			})
			result, err := callCollectReviews(t, ctx, fetcher, input, gate)
			if err != nil || !result.IsError {
				t.Fatalf("拒否=%+v %v", result, err)
			}

			result, err = callCollectReviews(t, context.Background(), fetcher, json.RawMessage(`{"app_id":440}`), gate)
			if err != nil || result.IsError {
				t.Fatalf("回数が消費された: %+v %v", result, err)
			}
		})
	}
}

func TestCollectionOptionsFailureDoesNotConsumeBudget(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input json.RawMessage
	}{
		{"不正な言語で条件を生成できなければ枠を消費しない", json.RawMessage(`{"app_id":440,"language":"Japanese"}`)},
		{"ページ件数が上限を超えて条件を生成できなければ枠を消費しない", json.RawMessage(`{"app_id":440,"page_size":101}`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gate := mcpdomain.NewLimiter(testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 1, MaxConcurrent: 1}), time.Now)
			calls := 0
			fetcher := testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				calls++
				return steamdomain.ReviewPage{}, nil
			})
			result, err := callCollectReviews(t, t.Context(), fetcher, tc.input, gate)
			if err != nil {
				t.Fatal(err)
			}

			out := testutil.DecodeOutput(t, result)
			if !result.IsError || out.ErrorCode != "invalid_input" || calls != 0 || out.Collection.AppID != 0 || out.Collection.Options.Language != "" || out.Collection.Options.PageSize != 0 {
				t.Fatalf("拒否=%+v calls=%d", out, calls)
			}

			result, err = callCollectReviews(t, t.Context(), fetcher, json.RawMessage(`{"app_id":440}`), gate)
			if err != nil || result.IsError || calls != 1 {
				t.Fatalf("後続の有効な入力=%+v err=%v calls=%d", result, err, calls)
			}
		})
	}
}
