package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMCPRoute(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method string
		path   string
		status int
		called bool
	}{
		{"POSTのMCPを委譲する", "POST", "/tools", 202, true},
		{"MCPのメソッド判定を委譲する", "GET", "/tools", 202, true},
		{"MCPの子パスを公開しない", "POST", "/tools/child", 404, false},
		{"末尾のスラッシュを公開しない", "POST", "/tools/", 404, false},
		{"旧パスを公開しない", "POST", "/mcp", 404, false},
		{"OAuthのmetadataを公開しない", "GET", "/.well-known/oauth-protected-resource/tools", 404, false},
		{"OAuthのmetadataへPOSTできない", "POST", "/.well-known/oauth-protected-resource/tools", 404, false},
		{"healthzはMCPに委譲しない", "GET", "/healthz", 200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			handler := newHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.URL.Path != "/tools" {
					t.Fatal(r.URL.Path)
				}
				w.WriteHeader(202)
			}))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status || called != tc.called {
				t.Fatalf("status=%d called=%v", w.Code, called)
			}
		})
	}
}
