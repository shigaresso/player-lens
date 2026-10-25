package adapter

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
		{"既存の無関係な経路を保持する", http.MethodGet, "/existing", false, http.StatusCreated, "existing"},
		{"未定義パスはhandlerを呼ばず404を返す", http.MethodGet, "/", false, http.StatusNotFound, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []string
			health := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls = append(calls, "health")
				w.WriteHeader(http.StatusNoContent)
			})
			var mcp http.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
			RegisterRoutes(mux, health, mcp)
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
