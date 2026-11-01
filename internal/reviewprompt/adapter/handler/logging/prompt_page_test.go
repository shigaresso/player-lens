package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"
)

func decodePageLog(t *testing.T, output string) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Fatalf("完了イベントは1件必要: %q", output)
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatal(err)
	}

	return record
}

func TestPromptPageStatus(t *testing.T) {
	for _, tc := range []struct {
		name           string
		status         int
		level, outcome string
	}{
		{"通常の応答", 200, "INFO", "success"},
		{"部分応答", 206, "INFO", "success"},
		{"未変更の応答", 304, "INFO", "success"},
		{"要求の拒否", 416, "WARN", "rejected"},
		{"内部の応答失敗", 500, "ERROR", "response_failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil)).With("service.name", "player-lens")
			PromptPage{Route: "/", Method: "GET", StatusCode: tc.status, Duration: 2500 * time.Microsecond}.Write(context.Background(), logger)
			record := decodePageLog(t, output.String())
			for key, want := range map[string]any{
				"event": "prompt_page.completed", "msg": "プロンプト作成画面の応答を処理した",
				"service.name": "player-lens", "level": tc.level, "outcome": tc.outcome,
				"http.route": "/", "http.request.method": "GET", "http.response.status_code": float64(tc.status), "duration_ms": 2.5,
			} {
				if record[key] != want {
					t.Errorf("%s=%v; want=%v", key, record[key], want)
				}
			}

			if len(record) != 10 || record["error_code"] != nil || record["cause_kind"] != nil || record["trace_id"] != nil {
				t.Fatalf("未定義の属性を記録した: %+v", record)
			}
		})
	}
}

func TestPromptPageWriteFailure(t *testing.T) {
	for _, tc := range []struct {
		name   string
		cause  error
		kind   string
		status int
	}{
		{"キャンセル", context.Canceled, "cancelled", 200},
		{"期限超過", context.DeadlineExceeded, "deadline_exceeded", 200},
		{"閉じたパイプ", io.ErrClosedPipe, "connection_closed", 200},
		{"閉じた接続", net.ErrClosed, "connection_closed", 200},
		{"切断", syscall.EPIPE, "connection_closed", 200},
		{"接続リセット", syscall.ECONNRESET, "connection_closed", 200},
		{"通信タイムアウト", &net.OpError{Op: "write", Net: "tcp", Err: os.ErrDeadlineExceeded}, "timeout", 200},
		{"ラップした失敗", fmt.Errorf("private diagnostic: %w", io.ErrClosedPipe), "connection_closed", 416},
		{"未知の失敗", errors.New("private diagnostic"), "unknown", 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			PromptPage{Route: "/", Method: "GET", StatusCode: tc.status, WriteErr: tc.cause}.Write(context.Background(), logger)
			record := decodePageLog(t, output.String())
			if record["level"] != "ERROR" || record["outcome"] != "write_failed" || record["error_code"] != "response_write_failed" || record["cause_kind"] != tc.kind || record["http.response.status_code"] != float64(tc.status) {
				t.Fatalf("失敗分類が異なる: %+v", record)
			}

			if len(record) != 11 || strings.Contains(output.String(), "private") || record["error"] != nil {
				t.Fatal("エラー本文または未定義の属性を記録した")
			}
		})
	}
}
