package adapter

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		status int
		body   string
	}{
		{"GETは正常応答を返す", http.MethodGet, "/healthz", http.StatusOK, "ok\n"},
		{"HEADは本文を返さない", http.MethodHead, "/healthz", http.StatusOK, ""},
		{"POSTは許可しない", http.MethodPost, "/healthz", http.StatusMethodNotAllowed, ""},
		{"未定義パスを公開しない", http.MethodGet, "/", http.StatusNotFound, ""},
		{"子パスをヘルスチェックに含めない", http.MethodGet, "/healthz/child", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			NewHandler(nil, slog.New(slog.NewJSONHandler(io.Discard, nil)), time.Now).ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
			if response.Code != tt.status {
				t.Fatalf("status=%d; want=%d", response.Code, tt.status)
			}

			if tt.status == http.StatusOK {
				if response.Body.String() != tt.body {
					t.Errorf("body=%q; want=%q", response.Body.String(), tt.body)
				}

				if got := response.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
					t.Errorf("Content-Type=%q", got)
				}
			}

			if tt.status == http.StatusMethodNotAllowed && response.Header().Get("Allow") != "GET, HEAD" {
				t.Errorf("Allow=%q", response.Header().Get("Allow"))
			}
		})
	}
}

type failedResponse struct {
	header   http.Header
	cause    error
	writes   int
	statuses []int
}

func (w *failedResponse) Header() http.Header  { return w.header }
func (w *failedResponse) WriteHeader(code int) { w.statuses = append(w.statuses, code) }
func (w *failedResponse) Write([]byte) (int, error) {
	w.writes++
	return 0, w.cause
}

type healthNetworkError struct{ timeout bool }

func (e healthNetworkError) Error() string   { return "secret: network details" }
func (e healthNetworkError) Timeout() bool   { return e.timeout }
func (e healthNetworkError) Temporary() bool { return false }

func TestHealthWriteFailure(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cause error
		kind  string
	}{
		{"既存の文字列原因は不明と分類する", errors.New("接続切断"), "unknown"},
		{"未知の原因の機密文字列を出力しない", errors.New("secret: connection details"), "unknown"},
		{"ラップしたpipe切断を識別する", fmt.Errorf("write: %w", io.ErrClosedPipe), "connection_closed"},
		{"閉じた接続を識別する", net.ErrClosed, "connection_closed"},
		{"broken pipeを識別する", syscall.EPIPE, "connection_closed"},
		{"接続のリセットを識別する", syscall.ECONNRESET, "connection_closed"},
		{"キャンセルを識別する", context.Canceled, "cancelled"},
		{"期限を識別する", context.DeadlineExceeded, "deadline_exceeded"},
		{"通信のtimeoutを識別する", healthNetworkError{timeout: true}, "timeout"},
		{"timeoutではない通信エラーは不明とする", healthNetworkError{}, "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			response := &failedResponse{header: make(http.Header), cause: tc.cause}
			NewHandler(nil, logger, time.Now).ServeHTTP(response,
				httptest.NewRequest(http.MethodGet, "/healthz?secret=value", nil))
			record := decodeHealthLog(t, output.String())
			for key, want := range map[string]any{
				"level": "ERROR", "msg": "ヘルスチェックを処理した", "event": "health_check.completed",
				"outcome": "write_failed", "error_code": "response_write_failed", "cause_kind": tc.kind,
				"http.request.method": "GET", "http.route": "/healthz", "http.response.status_code": float64(200),
			} {
				if record[key] != want {
					t.Errorf("%s=%v; want=%v", key, record[key], want)
				}
			}

			if response.writes != 1 || len(response.statuses) != 0 || response.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Fatalf("応答を再生成した: writes=%d status=%v header=%v", response.writes, response.statuses, response.header)
			}

			if strings.Contains(output.String(), "secret") || record["duration_ms"] == nil {
				t.Fatalf("機密情報または所要時間の契約が不正: %s", output.String())
			}
		})
	}
}
