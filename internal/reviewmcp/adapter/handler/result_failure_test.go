package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/handler/protocol"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectCombinedFailurePriority(t *testing.T) {
	internalFailure := errors.New("secret: upstream details")
	for _, tc := range []struct {
		name  string
		cause error
		code  string
	}{
		{"キャンセルと期限が重なる場合はキャンセルを表示する", errors.Join(context.Canceled, context.DeadlineExceeded, internalFailure), "cancelled"},
		{"期限と取得失敗が重なる場合は期限を表示する", errors.Join(context.DeadlineExceeded, internalFailure), "deadline_exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fetcher := testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				return steamdomain.ReviewPage{}, tc.cause
			})
			result, err := callCollectReviews(t, t.Context(), fetcher, json.RawMessage(`{"app_id":440}`), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
			if err != nil || result == nil || !result.IsError {
				t.Fatalf("result=%+v err=%v", result, err)
			}

			out := testutil.DecodeOutput(t, result)
			if out.ErrorCode != tc.code || out.Message == "" || out.Collection.StopReason != steamdomain.StopFailed {
				t.Fatal(out)
			}

			if !errors.Is(result.GetError(), tc.cause) || !errors.Is(result.GetError(), internalFailure) {
				t.Fatal("サーバー側の複合原因を失った")
			}

			data, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}

			if strings.Contains(string(data), "secret") {
				t.Fatal("内部の取得エラーを公開した")
			}
		})
	}
}

func TestCollectOversizedFailure(t *testing.T) {
	cause := errors.New("secret: collection failed after a large page")
	calls := 0
	fetcher := testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		calls++
		if calls == 1 {
			return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 1, Text: strings.Repeat("x", 70000)}}, NextCursor: "next"}, nil
		}

		return steamdomain.ReviewPage{}, cause
	})
	result, err := callCollectReviews(t, t.Context(), fetcher, json.RawMessage(`{"app_id":440,"max_reviews":2}`), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("result=%+v err=%v", result, err)
	}

	out := testutil.DecodeOutput(t, result)
	if calls != 2 || out.ErrorCode != "response_too_large" || out.ReturnedCount != 0 || out.OmittedReviews != 1 || len(out.Collection.Reviews) != 0 {
		t.Fatalf("calls=%d output=%+v", calls, out)
	}

	if out.Collection.AppID != 440 || out.Collection.Options.MaxReviews != 2 || out.Collection.PagesFetched != 1 || out.Collection.StopReason != steamdomain.StopFailed {
		t.Fatal("部分収集の条件と失敗状態を失った")
	}

	if !errors.Is(result.GetError(), protocol.ErrResponseTooLarge) || !errors.Is(result.GetError(), cause) {
		t.Fatalf("複合原因を失った: %v", result.GetError())
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}

	if len(data) > 128*1024 || strings.Contains(string(data), "secret") {
		t.Fatal("応答量または内部エラーの非公開の契約を満たさなかった")
	}
}
