//go:build integration

package main

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	serveradapter "github.com/shigaresso/player-lens/cmd/server/adapter"
	serverhandler "github.com/shigaresso/player-lens/cmd/server/adapter/handler"
)

func TestPromptBuilderWithHTTP(t *testing.T) {
	page, err := serverhandler.NewPromptBuilderHandler()
	if err != nil {
		t.Fatal(err)
	}

	router := http.NewServeMux()
	health := serverhandler.NewHealthHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), time.Now)
	serveradapter.RegisterRoutes(router, health, nil, page)
	url, client, _, _, _ := startTestServer(t, router, time.Second)
	for _, tc := range []struct {
		name, method, path string
		status             int
	}{
		{"GETで埋め込み画面を返す", "GET", "/", 200},
		{"HEADは本文を返さない", "HEAD", "/", 200},
		{"画面へPOSTさせない", "POST", "/", 405},
		{"未定義パスは画面にしない", "GET", "/missing", 404},
		{"生存確認を維持する", "GET", "/healthz", 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			request, err := http.NewRequestWithContext(t.Context(), tc.method, url+tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}

			body, readErr := io.ReadAll(response.Body)
			closeErr := response.Body.Close()
			if readErr != nil || closeErr != nil || response.StatusCode != tc.status {
				t.Fatalf("status=%d read=%v close=%v", response.StatusCode, readErr, closeErr)
			}

			if tc.path == "/" && tc.method == "GET" && !strings.Contains(string(body), `id="prompt-form"`) {
				t.Fatal("作成画面を返さなかった")
			}

			if tc.method == "HEAD" && len(body) != 0 {
				t.Fatal("HEAD に本文を返した")
			}
		})
	}
}
