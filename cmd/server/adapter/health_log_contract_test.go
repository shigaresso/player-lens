package adapter

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type healthContextKey struct{}

type healthContextHandler struct {
	slog.Handler
	seen []context.Context
}

func (h *healthContextHandler) Handle(ctx context.Context, record slog.Record) error {
	h.seen = append(h.seen, ctx)
	return h.Handler.Handle(ctx, record)
}

type failedHealthLogWriter struct{ calls int }

func (w *failedHealthLogWriter) Write([]byte) (int, error) {
	w.calls++
	return 0, io.ErrClosedPipe
}

func TestNewHealthHandlerDoesNotProcessRequest(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	clockCalls := 0
	now := func() time.Time {
		clockCalls++
		return time.Unix(100, 0)
	}
	handler := NewHealthHandler(logger, now)
	if handler == nil || clockCalls != 0 || output.Len() != 0 {
		t.Fatalf("handlerの生成時に処理を実行した: handler=%v clockCalls=%d log=%q", handler, clockCalls, output.String())
	}
}

func TestHealthLogFailurePreservesResponse(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		body   string
	}{
		{"ログ出力が失敗してもGETの応答を返す", http.MethodGet, "ok\n"},
		{"ログ出力が失敗してもHEADの本文を出さない", http.MethodHead, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := &failedHealthLogWriter{}
			logger := slog.New(slog.NewJSONHandler(output, nil))
			response := httptest.NewRecorder()
			NewHealthHandler(logger, time.Now).ServeHTTP(response, httptest.NewRequest(tc.method, "/healthz", nil))
			if response.Code != http.StatusOK || response.Body.String() != tc.body || response.Header().Get("Content-Type") != "text/plain; charset=utf-8" || output.calls != 1 {
				t.Fatalf("ログの失敗で応答を変更した: status=%d body=%q header=%v logCalls=%d", response.Code, response.Body.String(), response.Header(), output.calls)
			}
		})
	}
}

func TestHealthLogReceivesRequestContext(t *testing.T) {
	for _, tc := range []struct {
		name      string
		cancelled bool
	}{
		{"呼び出し元のcontextと値をログへ渡す", false},
		{"キャンセル済みでも同じcontextをログへ渡す", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.WithValue(t.Context(), healthContextKey{}, "trace-context"))
			t.Cleanup(cancel)
			if tc.cancelled {
				cancel()
			}

			handler := &healthContextHandler{Handler: slog.NewJSONHandler(io.Discard, nil)}
			logger := slog.New(handler)
			request := httptest.NewRequest(http.MethodGet, "/healthz", nil).WithContext(ctx)
			NewHealthHandler(logger, time.Now).ServeHTTP(httptest.NewRecorder(), request)
			if len(handler.seen) != 1 || handler.seen[0] != ctx || handler.seen[0].Value(healthContextKey{}) != "trace-context" {
				t.Fatalf("request contextを引き継がなかった: %v", handler.seen)
			}
		})
	}
}

func TestOtherRoutesDoNotEmitHealthLog(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		path   string
		status int
	}{
		{"未許可メソッドは生存確認イベントに含めない", http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
		{"未定義パスは生存確認イベントに含めない", http.MethodGet, "/", http.StatusNotFound},
		{"子パスは生存確認イベントに含めない", http.MethodGet, "/healthz/child", http.StatusNotFound},
		{"MCPへの委譲は生存確認イベントに含めない", http.MethodPost, "/tools", http.StatusNoContent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil))
			mcp := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
			response := httptest.NewRecorder()
			router := http.NewServeMux()
			RegisterRoutes(router, NewHealthHandler(logger, time.Now), mcp)
			router.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			if response.Code != tc.status || output.Len() != 0 {
				t.Fatalf("対象外の経路で契約を変更した: status=%d log=%q", response.Code, output.String())
			}
		})
	}
}
