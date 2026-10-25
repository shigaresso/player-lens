package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

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
			fetcher := fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				return steamdomain.ReviewPage{}, tc.cause
			})
			result, err := collect(t.Context(), fetcher, json.RawMessage(`{"app_id":440}`), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
			if err != nil || result == nil || !result.IsError {
				t.Fatalf("result=%+v err=%v", result, err)
			}

			out := decodeOutput(t, result)
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
	fetcher := fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		calls++
		if calls == 1 {
			return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 1, Text: strings.Repeat("x", 70000)}}, NextCursor: "next"}, nil
		}

		return steamdomain.ReviewPage{}, cause
	})
	result, err := collect(t.Context(), fetcher, json.RawMessage(`{"app_id":440,"max_reviews":2}`), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("result=%+v err=%v", result, err)
	}

	out := decodeOutput(t, result)
	if calls != 2 || out.ErrorCode != "response_too_large" || out.ReturnedCount != 0 || out.OmittedReviews != 1 || len(out.Collection.Reviews) != 0 {
		t.Fatalf("calls=%d output=%+v", calls, out)
	}

	if out.Collection.AppID != 440 || out.Collection.Options.MaxReviews != 2 || out.Collection.PagesFetched != 1 || out.Collection.StopReason != steamdomain.StopFailed {
		t.Fatal("部分収集の条件と失敗状態を失った")
	}

	if !errors.Is(result.GetError(), errResponseTooLarge) || !errors.Is(result.GetError(), cause) {
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

func TestEncodeCollectionResultEncodingFailureCause(t *testing.T) {
	cause := errors.New("secret: upstream details")
	for _, tc := range []struct {
		name       string
		collection steamdomain.Collection
		wantErr    error
	}{
		{"省略後も大きい場合に変換と収集の原因を保持する", steamdomain.Collection{Options: steamdomain.CollectionOptionsConfig{Language: strings.Repeat("x", maxResultBytes+1)}}, errResponseTooLarge},
		{"JSON変換が失敗した場合も変換と収集の原因を保持する", steamdomain.Collection{Options: steamdomain.CollectionOptionsConfig{ReviewType: steamdomain.ReviewType(3)}}, steamdomain.ErrInvalidArgument},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result, err := encodeCollectionResult(tc.collection, cause)
			if result != nil || !errors.Is(err, tc.wantErr) {
				t.Fatalf("result=%+v error=%v", result, err)
			}

			if !errors.Is(err, cause) {
				t.Fatalf("収集の原因を失った: %v", err)
			}

			if strings.Contains(err.Error(), "secret") {
				t.Fatalf("内部の収集原因を表示文言に含めた: %v", err)
			}
		})
	}
}
