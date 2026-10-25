//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	serveradapter "github.com/shigaresso/player-lens/cmd/server/adapter"
)

type healthLogBuffer struct {
	mu   sync.Mutex
	data bytes.Buffer
}

func (b *healthLogBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.data.Write(data)
}

func (b *healthLogBuffer) snapshot() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	return bytes.Clone(b.data.Bytes())
}

func TestHealthStructuredLogWithHTTP(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		body   string
	}{
		{"実HTTPのGETで応答とJSONを確認する", http.MethodGet, "ok\n"},
		{"実HTTPのHEADで本文なしとJSONを確認する", http.MethodHead, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := &healthLogBuffer{}
			logger := slog.New(slog.NewJSONHandler(output, nil)).With("service.name", "player-lens")
			url, client, _, _, _ := startTestServer(t, serveradapter.NewHandler(nil, logger, time.Now), time.Second)
			request, err := http.NewRequestWithContext(t.Context(), tc.method, url+"/healthz", nil)
			if err != nil {
				t.Fatal(err)
			}

			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}

			body, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if readErr != nil || closeErr != nil || response.StatusCode != http.StatusOK || string(body) != tc.body || response.Header.Get("Content-Type") != "text/plain; charset=utf-8" {
				t.Fatalf("HTTPの契約を変更した: status=%d body=%q read=%v close=%v", response.StatusCode, body, readErr, closeErr)
			}

			var record map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(output.snapshot()), &record); err != nil {
				t.Fatalf("完了ログが単一のJSONではない: %v", err)
			}

			for key, want := range map[string]any{
				"event": "health_check.completed", "level": "INFO", "service.name": "player-lens",
				"http.request.method": tc.method, "outcome": "success",
			} {
				if record[key] != want {
					t.Errorf("%s=%v; want=%v", key, record[key], want)
				}
			}

			if duration, ok := record["duration_ms"].(float64); !ok || duration < 0 {
				t.Fatalf("所要時間が不正: %v", record["duration_ms"])
			}
		})
	}
}
