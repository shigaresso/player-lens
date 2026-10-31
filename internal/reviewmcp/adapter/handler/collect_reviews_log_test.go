package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type collectionContextKey struct{}

type collectionLogHandler struct {
	slog.Handler
	calls int
	value any
	err   error
}

func (h *collectionLogHandler) Handle(ctx context.Context, record slog.Record) error {
	h.calls++
	h.value, h.err = ctx.Value(collectionContextKey{}), ctx.Err()
	return h.Handler.Handle(ctx, record)
}

func TestCollectReviewsLog(t *testing.T) {
	for _, tc := range []struct {
		name      string
		input     string
		cause     error
		partial   bool
		oversized bool
		rejected  bool
		cancelled bool
		code      string
		kind      string
		calls     int
		count     int
	}{
		{name: "成功", input: `{"app_id":440,"max_reviews":1}`, calls: 1, count: 1},
		{name: "不正入力", input: `{"app_id":440,"page_size":"private input"}`, code: "invalid_input", kind: "json_type"},
		{name: "受付拒否", input: `{"app_id":440}`, rejected: true, code: "daily_limit_reached", kind: "usage_limit"},
		{name: "開始前キャンセル", input: `{"app_id":440}`, cancelled: true, code: "cancelled", kind: "cancelled"},
		{name: "取得失敗", input: `{"app_id":440}`, cause: errors.New("private upstream"), code: "fetch_failed", kind: "unknown", calls: 1},
		{name: "部分失敗", input: `{"app_id":440,"max_reviews":2}`, partial: true, cause: errors.New("private upstream"), code: "fetch_failed", kind: "unknown", calls: 2, count: 1},
		{name: "期限超過", input: `{"app_id":440}`, cause: context.DeadlineExceeded, code: "deadline_exceeded", kind: "deadline_exceeded", calls: 1},
		{name: "応答の本文省略", input: `{"app_id":440,"max_reviews":1}`, oversized: true, code: "response_too_large", kind: "response_too_large", calls: 1, count: 1},
	} {
		t.Run(tc.name+"の応答と計測とcontextを1回記録する", func(t *testing.T) {
			var output bytes.Buffer
			probe := &collectionLogHandler{Handler: slog.NewJSONHandler(&output, nil)}
			ctx, cancel := context.WithCancel(context.WithValue(t.Context(), collectionContextKey{}, "trace-context"))
			t.Cleanup(cancel)
			if tc.cancelled {
				cancel()
			}

			gate := mcpdomain.NewLimiter(testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 1, MaxConcurrent: 1}), time.Now)
			if tc.rejected {
				release, _ := gate.Acquire()
				release()
			}

			clockCalls, fetchCalls := 0, 0
			now := func() time.Time {
				clockCalls++
				return time.Unix(100, int64(clockCalls)*int64(1250*time.Microsecond))
			}
			handler := NewCollectReviewsHandler(testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				fetchCalls++
				if tc.cause != nil && (!tc.partial || fetchCalls > 1) {
					return steamdomain.ReviewPage{}, tc.cause
				}

				text := "private review"
				if tc.oversized {
					text = strings.Repeat(text, 10000)
				}

				return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 1, Text: text}}, NextCursor: "private cursor"}, nil
			}), ctx, gate, slog.New(probe), now)
			if clockCalls != 0 || output.Len() != 0 {
				t.Fatal("構築時に計測かログを実行した")
			}

			result, err := handler(ctx, &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(tc.input)}})
			if err != nil || result == nil || result.IsError != (tc.code != "") || fetchCalls != tc.calls {
				t.Fatalf("応答を変更した: result=%+v err=%v fetchCalls=%d", result, err, fetchCalls)
			}

			out := testutil.DecodeOutput(t, result)
			if out.ErrorCode != tc.code || probe.calls != 1 || clockCalls != 2 || probe.value != "trace-context" || (probe.err != nil) != tc.cancelled {
				t.Fatalf("計測・context・公開コードの契約が異なる: %+v probe=%+v clockCalls=%d", out, probe, clockCalls)
			}

			var record map[string]any
			if err := json.Unmarshal(output.Bytes(), &record); err != nil {
				t.Fatal(err)
			}

			if record["duration_ms"] != 1.25 || record["event"] != "review_collection.completed" || strings.Contains(output.String(), "private") {
				t.Fatal("計測または非漏出の契約を満たさなかった")
			}

			if tc.code != "" && (record["error_code"] != tc.code || record["cause_kind"] != tc.kind) {
				t.Fatalf("公開分類とログが異なる: %+v", record)
			}

			if tc.code == "invalid_input" || tc.rejected {
				if _, exists := record["collected_count"]; exists {
					t.Fatal("収集していない件数を記録した")
				}
			} else if record["collected_count"] != float64(tc.count) {
				t.Fatalf("省略前の収集件数を記録しなかった: %+v", record)
			}
		})
	}
}

type failedCollectionLogWriter struct{ calls int }

func (w *failedCollectionLogWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, io.ErrClosedPipe
}

func TestCollectReviewsLogWriteFailure(t *testing.T) {
	writer := &failedCollectionLogWriter{}
	gate := mcpdomain.NewLimiter(testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 10, MaxConcurrent: 1}), time.Now)
	fetchCalls := 0
	fetcher := testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		fetchCalls++
		return steamdomain.ReviewPage{}, nil
	})
	handler := NewCollectReviewsHandler(fetcher, t.Context(), gate, slog.New(slog.NewJSONHandler(writer, nil)), time.Now)
	for _, name := range []string{"ログ失敗でも応答を維持する", "ログ失敗後も枠を再利用できる"} {
		t.Run(name, func(t *testing.T) {
			result, err := handler(t.Context(), &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Arguments: json.RawMessage(`{"app_id":440}`)}})
			if err != nil || result == nil || result.IsError {
				t.Fatalf("ログ失敗で応答を変更した: result=%+v err=%v", result, err)
			}
		})
	}

	if writer.calls != 2 || fetchCalls != 2 {
		t.Fatalf("ログの再試行か枠の未返却が発生した: writes=%d fetches=%d", writer.calls, fetchCalls)
	}
}
