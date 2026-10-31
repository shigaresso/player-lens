package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/handler/protocol"
	mcpapplication "github.com/shigaresso/player-lens/internal/reviewmcp/application"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func collected(cause error) *mcpapplication.Result {
	return &mcpapplication.Result{Collection: steamdomain.Collection{
		AppID: 440, Reviews: []steamdomain.Review{{Text: "private review"}}, PagesFetched: 2, StopReason: steamdomain.StopReviewLimit,
	}, Err: cause}
}

func TestReviewCollectionWrite(t *testing.T) {
	var syntaxCause *json.SyntaxError
	if err := json.Unmarshal([]byte(`{"private":`), new(any)); !errors.As(err, &syntaxCause) {
		t.Fatal("構文エラーを準備できなかった")
	}

	var typeCause *json.UnmarshalTypeError
	if err := json.Unmarshal([]byte(`"private"`), new(int)); !errors.As(err, &typeCause) {
		t.Fatal("型エラーを準備できなかった")
	}

	for _, tc := range []struct {
		name    string
		entry   ReviewCollection
		level   string
		outcome string
		stage   string
		code    string
		kind    string
	}{
		{"成功", ReviewCollection{Result: collected(nil)}, "INFO", "success", "", "", ""},
		{"構文不正", ReviewCollection{InputErr: syntaxCause}, "INFO", "rejected", "input", "invalid_input", "json_syntax"},
		{"型不正", ReviewCollection{InputErr: typeCause}, "INFO", "rejected", "input", "invalid_input", "json_type"},
		{"条件不正", ReviewCollection{InputErr: fmt.Errorf("private input: %w", steamdomain.ErrInvalidArgument)}, "INFO", "rejected", "input", "invalid_input", "invalid_argument"},
		{"日次拒否", ReviewCollection{Result: &mcpapplication.Result{Rejection: mcpdomain.DailyLimitReached}}, "INFO", "rejected", "collection", "daily_limit_reached", "usage_limit"},
		{"分次拒否", ReviewCollection{Result: &mcpapplication.Result{Rejection: mcpdomain.MinuteLimitReached}}, "INFO", "rejected", "collection", "rate_limited", "usage_limit"},
		{"同時数拒否", ReviewCollection{Result: &mcpapplication.Result{Rejection: mcpdomain.ConcurrentLimitReached}}, "INFO", "rejected", "collection", "busy", "usage_limit"},
		{"キャンセル", ReviewCollection{Result: collected(context.Canceled)}, "INFO", "cancelled", "collection", "cancelled", "cancelled"},
		{"期限超過", ReviewCollection{Result: collected(context.DeadlineExceeded)}, "WARN", "failed", "collection", "deadline_exceeded", "deadline_exceeded"},
		{"HTTP ステータス", ReviewCollection{Result: collected(&steamadapter.HTTPStatusError{StatusCode: 503})}, "WARN", "failed", "collection", "fetch_failed", "http_status"},
		{"カーソル循環", ReviewCollection{Result: collected(steamdomain.ErrCursorCycle)}, "WARN", "failed", "collection", "fetch_failed", "cursor_cycle"},
		{"Steam 応答サイズ超過", ReviewCollection{Result: collected(steamadapter.ErrResponseTooLarge)}, "WARN", "failed", "collection", "fetch_failed", "upstream_response_too_large"},
		{"Steam 応答不正", ReviewCollection{Result: collected(steamadapter.ErrInvalidResponse)}, "WARN", "failed", "collection", "fetch_failed", "invalid_response"},
		{"通信のタイムアウト", ReviewCollection{Result: collected(&net.DNSError{Err: "private", Name: "private.invalid", IsTimeout: true})}, "WARN", "failed", "collection", "fetch_failed", "timeout"},
		{"未知の通信失敗", ReviewCollection{Result: collected(&url.Error{Op: "GET", URL: "https://private.invalid/?token=private", Err: errors.New("private cause")})}, "WARN", "failed", "collection", "fetch_failed", "unknown"},
		{"本文省略", ReviewCollection{Result: collected(nil), ResponseTooLarge: true}, "WARN", "failed", "response_encode", "response_too_large", "response_too_large"},
		{"応答生成不能", ReviewCollection{Result: collected(nil), ResponseErr: errors.New("private encoding")}, "ERROR", "failed", "response_encode", "response_encoding_failed", "unknown"},
		{"省略後も応答サイズ超過", ReviewCollection{Result: collected(nil), ResponseErr: protocol.ErrResponseTooLarge}, "ERROR", "failed", "response_encode", "response_too_large", "response_too_large"},
		{"入力失敗より応答生成不能を優先", ReviewCollection{InputErr: syntaxCause, ResponseErr: errors.New("private encoding")}, "ERROR", "failed", "response_encode", "response_encoding_failed", "unknown"},
		{"panic", ReviewCollection{Panicked: true, Stage: "response_encode", Result: collected(nil)}, "ERROR", "failed", "response_encode", "panic", "panic"},
	} {
		t.Run(tc.name+"の固定属性と安全な分類を1回記録する", func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil)).With("service.name", "player-lens")
			entry := tc.entry
			entry.Duration = 1250 * time.Microsecond
			if entry.InputErr == nil {
				entry.AppID = 440
			}

			entry.Write(t.Context(), logger)
			if bytes.Count(output.Bytes(), []byte("\n")) != 1 || strings.Contains(output.String(), "private") {
				t.Fatal("ログの回数か非公開情報の契約を破った")
			}

			var record map[string]any
			if err := json.Unmarshal(output.Bytes(), &record); err != nil {
				t.Fatal(err)
			}

			for key, want := range map[string]any{
				"event": "review_collection.completed", "service.name": "player-lens", "duration_ms": 1.25,
				"level": tc.level, "outcome": tc.outcome,
			} {
				if record[key] != want {
					t.Errorf("%s=%v; want=%v", key, record[key], want)
				}
			}

			for key, want := range map[string]string{"failure_stage": tc.stage, "error_code": tc.code, "cause_kind": tc.kind} {
				if want == "" {
					if _, exists := record[key]; exists {
						t.Errorf("成功時に %s を付けた", key)
					}
				} else if record[key] != want {
					t.Errorf("%s=%v; want=%v", key, record[key], want)
				}
			}

			if entry.AppID == 0 {
				if _, exists := record["app_id"]; exists {
					t.Error("入力解析失敗時に app_id を記録した")
				}
			} else if record["app_id"] != float64(440) {
				t.Error("対象作品を記録しなかった")
			}

			if entry.Result == nil || entry.Result.Rejection != mcpdomain.NotRejected {
				for _, key := range []string{"collected_count", "pages_fetched", "stop_reason"} {
					if _, exists := record[key]; exists {
						t.Errorf("収集結果がないのに %s を記録した", key)
					}
				}
			} else if record["collected_count"] != float64(1) || record["pages_fetched"] != float64(2) || record["stop_reason"] != "review_limit" {
				t.Fatal("省略前の収集結果を保持しなかった")
			}
		})
	}
}
