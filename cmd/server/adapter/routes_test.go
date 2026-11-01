package adapter

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	serverhandler "github.com/shigaresso/player-lens/cmd/server/adapter/handler"
)

func TestPromptBuilderRoutes(t *testing.T) {
	for _, tc := range []struct {
		name, method, path string
		status             int
		calls              int
	}{
		{"GETは画面に委譲する", "GET", "/", http.StatusNoContent, 1},
		{"HEADは画面に委譲する", "HEAD", "/", http.StatusNoContent, 1},
		{"POSTは許可しない", "POST", "/", http.StatusMethodNotAllowed, 0},
		{"任意の子パスを画面にしない", "GET", "/unknown", http.StatusNotFound, 0},
		{"旧MCPパスを画面にしない", "GET", "/mcp", http.StatusNotFound, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			page := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.WriteHeader(http.StatusNoContent)
			})
			mux := http.NewServeMux()
			RegisterRoutes(mux, serverhandler.NewHealthHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), time.Now), nil, page)
			if calls != 0 {
				t.Fatal("登録時に画面を実行した")
			}

			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			if response.Code != tc.status || calls != tc.calls {
				t.Fatalf("status=%d calls=%d", response.Code, calls)
			}
		})
	}
}

func TestRegisterRoutes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		method     string
		path       string
		withoutMCP bool
		status     int
		called     string
	}{
		{"GETは渡したhealth handlerを呼ぶ", http.MethodGet, "/healthz", false, http.StatusNoContent, "health"},
		{"HEADは渡したhealth handlerを呼ぶ", http.MethodHead, "/healthz", false, http.StatusNoContent, "health"},
		{"未許可メソッドはhandlerを呼ばず405を返す", http.MethodPost, "/healthz", false, http.StatusMethodNotAllowed, ""},
		{"healthzの子パスは登録しない", http.MethodGet, "/healthz/child", false, http.StatusNotFound, ""},
		{"POSTは渡したMCP handlerを呼ぶ", http.MethodPost, "/tools", false, http.StatusAccepted, "mcp"},
		{"MCPのメソッド判定をhandlerへ委譲する", http.MethodGet, "/tools", false, http.StatusAccepted, "mcp"},
		{"MCP未指定ならtoolsを登録しない", http.MethodPost, "/tools", true, http.StatusNotFound, ""},
		{"MCPの子パスは登録しない", http.MethodPost, "/tools/child", false, http.StatusNotFound, ""},
		{"MCPの末尾スラッシュは登録しない", http.MethodPost, "/tools/", false, http.StatusNotFound, ""},
		{"旧MCPパスは登録しない", http.MethodPost, "/mcp", false, http.StatusNotFound, ""},
		{"OAuthのmetadataへGETできない", http.MethodGet, "/.well-known/oauth-protected-resource/tools", false, http.StatusNotFound, ""},
		{"OAuthのmetadataへPOSTできない", http.MethodPost, "/.well-known/oauth-protected-resource/tools", false, http.StatusNotFound, ""},
		{"既存の無関係な経路を保持する", http.MethodGet, "/existing", false, http.StatusCreated, "existing"},
		{"未定義パスはhandlerを呼ばず404を返す", http.MethodGet, "/", false, http.StatusNotFound, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			health := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls = append(calls, "health")
				w.WriteHeader(http.StatusNoContent)
			})
			var mcp http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/tools" {
					t.Fatalf("MCPへ渡したパスが異なる: %q", r.URL.Path)
				}

				calls = append(calls, "mcp")
				w.WriteHeader(http.StatusAccepted)
			})
			if tc.withoutMCP {
				mcp = nil
			}

			mux := http.NewServeMux()
			mux.HandleFunc("/existing", func(w http.ResponseWriter, _ *http.Request) {
				calls = append(calls, "existing")
				w.WriteHeader(http.StatusCreated)
			})
			RegisterRoutes(mux, health, mcp, nil)
			if len(calls) != 0 {
				t.Fatalf("登録時にhandlerを実行した: %v", calls)
			}

			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			if response.Code != tc.status {
				t.Fatalf("status=%d; want=%d", response.Code, tc.status)
			}

			if tc.called == "" {
				if len(calls) != 0 {
					t.Fatalf("対象外のhandlerを呼んだ: %v", calls)
				}
			} else if len(calls) != 1 || calls[0] != tc.called {
				t.Fatalf("指定されたhandlerへ一度委譲しなかった: %v; want=%s", calls, tc.called)
			}

			if tc.status == http.StatusMethodNotAllowed && response.Header().Get("Allow") != "GET, HEAD" {
				t.Errorf("Allow=%q", response.Header().Get("Allow"))
			}
		})
	}
}

func TestHealthRoutes(t *testing.T) {
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
			health := serverhandler.NewHealthHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), time.Now)
			router := http.NewServeMux()
			RegisterRoutes(router, health, nil, nil)
			router.ServeHTTP(response, httptest.NewRequest(tt.method, tt.path, nil))
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
			RegisterRoutes(router, serverhandler.NewHealthHandler(logger, time.Now), mcp, nil)
			router.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))
			if response.Code != tc.status || output.Len() != 0 {
				t.Fatalf("対象外の経路で契約を変更した: status=%d log=%q", response.Code, output.String())
			}
		})
	}
}
