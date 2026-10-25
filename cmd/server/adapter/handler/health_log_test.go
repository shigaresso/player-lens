package handler

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func decodeHealthLog(t *testing.T, output string) map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Fatalf("完了ログは1件必要: %q", output)
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("JSONのログではない: %v: %q", err, output)
	}

	return record
}

func TestHealthCompletionLog(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		body   string
	}{
		{"GETの応答と完了ログを返す", http.MethodGet, "ok\n"},
		{"HEADは本文なしの応答と完了ログを返す", http.MethodHead, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&output, nil)).With("service.name", "player-lens")
			start := time.Unix(100, 0)
			calls := 0
			now := func() time.Time {
				calls++
				if calls == 1 {
					return start
				}

				return start.Add(2500 * time.Microsecond)
			}
			request := httptest.NewRequest(tc.method, "/healthz?token=secret", nil)
			request.Header.Set("Authorization", "Bearer secret")
			response := httptest.NewRecorder()
			NewHealthHandler(logger, now).ServeHTTP(response, request)
			if response.Code != http.StatusOK || response.Body.String() != tc.body || response.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Fatalf("応答契約を変更した: status=%d body=%q header=%v", response.Code, response.Body.String(), response.Header())
			}

			record := decodeHealthLog(t, output.String())
			for key, want := range map[string]any{
				"level": "INFO", "msg": "ヘルスチェックを処理した", "service.name": "player-lens",
				"event": "health_check.completed", "http.request.method": tc.method, "http.route": "/healthz",
				"http.response.status_code": float64(200), "duration_ms": 2.5, "outcome": "success",
			} {
				if record[key] != want {
					t.Errorf("%s=%v; want=%v", key, record[key], want)
				}
			}

			stamp, ok := record["time"].(string)
			if _, err := time.Parse(time.RFC3339Nano, stamp); !ok || err != nil || calls != 2 {
				t.Fatalf("時刻と所要時間の取得が不正: time=%v calls=%d", record["time"], calls)
			}

			if strings.Contains(output.String(), "secret") || record["error_code"] != nil || record["trace_id"] != nil {
				t.Fatalf("機密情報または未発生の情報を出力した: %s", output.String())
			}
		})
	}
}
