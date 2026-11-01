package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type promptResponseWriter struct {
	*httptest.ResponseRecorder
	cause  error
	writes int
}

func (w *promptResponseWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.cause != nil {
		return 0, w.cause
	}

	return w.ResponseRecorder.Write(data)
}

func TestPromptPageCompletionLog(t *testing.T) {
	for _, tc := range []struct {
		name, method, header, value string
		status                      int
		cause                       error
		level, outcome              string
	}{
		{"GETの応答完了", "GET", "", "", 200, nil, "INFO", "success"},
		{"HEADの応答完了", "HEAD", "", "", 200, nil, "INFO", "success"},
		{"部分応答の完了", "GET", "Range", "bytes=0-3", 206, nil, "INFO", "success"},
		{"未変更応答の完了", "GET", "If-None-Match", "*", 304, nil, "INFO", "success"},
		{"範囲外の拒否", "GET", "Range", "bytes=99999999-", 416, nil, "WARN", "rejected"},
		{"切断での書き込み失敗", "GET", "", "", 200, io.ErrClosedPipe, "ERROR", "write_failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil)).With("service.name", "player-lens")
			calls := 0
			start := time.Unix(100, 0)
			now := func() time.Time {
				calls++
				return start.Add(time.Duration(calls-1) * 2500 * time.Microsecond)
			}
			handler, err := NewPromptBuilderHandler(logger, now)
			if err != nil {
				t.Fatal(err)
			}

			if calls != 0 || output.Len() != 0 {
				t.Fatal("生成時に計測またはログを実行した")
			}

			request := httptest.NewRequest(tc.method, "/?private=private-input", nil)
			request.Header.Set("Authorization", "private-header")
			request.Header.Set("User-Agent", "private-client")
			if tc.header != "" {
				request.Header.Set(tc.header, tc.value)
			}

			response := &promptResponseWriter{ResponseRecorder: httptest.NewRecorder(), cause: tc.cause}
			handler.ServeHTTP(response, request)
			var record map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(output.Bytes()), &record); err != nil {
				t.Fatalf("単一のJSONイベントではない: %v", err)
			}

			for key, want := range map[string]any{
				"event": "prompt_page.completed", "http.route": "/", "http.request.method": tc.method,
				"http.response.status_code": float64(tc.status), "duration_ms": 2.5,
				"level": tc.level, "outcome": tc.outcome, "service.name": "player-lens",
			} {
				if record[key] != want {
					t.Errorf("%s=%v; want=%v", key, record[key], want)
				}
			}

			if response.Code != tc.status || calls != 2 || strings.Contains(output.String(), "private") || record["trace_id"] != nil {
				t.Fatalf("応答・計測・非漏出の契約が異なる: status=%d calls=%d record=%+v", response.Code, calls, record)
			}

			if tc.method == "HEAD" && (response.Body.Len() != 0 || response.writes != 0) {
				t.Fatal("HEADで本文を書き込んだ")
			}

			if tc.cause != nil && (record["cause_kind"] != "connection_closed" || record["error_code"] != "response_write_failed" || response.writes != 1) {
				t.Fatal("失敗原因を保持しないか再書き込みした")
			}
		})
	}
}

type promptContextHandler struct {
	slog.Handler
	want  context.Context
	calls int
	t     *testing.T
}

func (h *promptContextHandler) Handle(ctx context.Context, record slog.Record) error {
	h.calls++
	if ctx != h.want {
		h.t.Error("リクエストの context を置き換えた")
	}

	return h.Handler.Handle(ctx, record)
}

func TestPromptPageLogContext(t *testing.T) {
	for _, state := range []string{"有効なcontext", "キャンセル済みcontext", "期限超過context"} {
		t.Run(state, func(t *testing.T) {
			ctx := t.Context()
			switch state {
			case "キャンセル済みcontext":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "期限超過context":
				var cancel context.CancelFunc
				ctx, cancel = context.WithDeadline(ctx, time.Unix(0, 0))
				defer cancel()
			}

			recorder := &promptContextHandler{Handler: slog.NewJSONHandler(io.Discard, nil), want: ctx, t: t}
			handler, err := NewPromptBuilderHandler(slog.New(recorder), time.Now)
			if err != nil {
				t.Fatal(err)
			}

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil).WithContext(ctx))
			if recorder.calls != 1 {
				t.Fatalf("context を使った記録回数=%d", recorder.calls)
			}
		})
	}
}

type brokenPromptLogWriter struct{}

func (brokenPromptLogWriter) Write([]byte) (int, error) {
	return 0, errors.New("private logging failure")
}

func TestPromptPageLogFailureDoesNotChangeResponse(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cause error
	}{
		{"正常応答は変えない", nil},
		{"既存の書き込み失敗で再送しない", io.ErrClosedPipe},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, err := NewPromptBuilderHandler(slog.New(slog.NewJSONHandler(brokenPromptLogWriter{}, nil)), time.Now)
			if err != nil {
				t.Fatal(err)
			}

			response := &promptResponseWriter{ResponseRecorder: httptest.NewRecorder(), cause: tc.cause}
			handler.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))
			if response.Code != 200 || response.writes != 1 {
				t.Fatal("ログの失敗で応答を変更・再送した")
			}

			if tc.cause == nil && !strings.Contains(response.Body.String(), `id="prompt-form"`) {
				t.Fatal("画面を返さなかった")
			}
		})
	}
}

type panickingPromptResponse struct{ *httptest.ResponseRecorder }

func (panickingPromptResponse) Write([]byte) (int, error) {
	panic("private response panic")
}

func TestPromptPagePanicPropagates(t *testing.T) {
	var output bytes.Buffer
	handler, err := NewPromptBuilderHandler(slog.New(slog.NewJSONHandler(&output, nil)), time.Now)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if recover() != "private response panic" || output.Len() != 0 {
			t.Error("panic を回復またはログへ転載した")
		}
	}()

	handler.ServeHTTP(panickingPromptResponse{httptest.NewRecorder()}, httptest.NewRequest("GET", "/", nil))
}
