package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	accesshandler "github.com/shigaresso/player-lens/internal/mcpaccess/adapter/handler"
	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

func TestProtect(t *testing.T) {
	for _, tc := range []struct {
		name, publicURL, host, origin, forwarded, fetchSite string
		status                                              int
	}{
		{name: "認証なしの公開Hostを許可する", host: "reviews.example", status: 202},
		{name: "大文字と既定443を正規化する", host: "REVIEWS.EXAMPLE:443", status: 202},
		{name: "localhostを許可する", host: "localhost:8080", status: 202},
		{name: "IPv4ループバックを許可する", host: "127.0.0.1:8080", status: 202},
		{name: "IPv6ループバックを許可する", host: "[::1]:8080", status: 202},
		{name: "一致するOriginを許可する", host: "reviews.example", origin: "https://reviews.example", status: 202},
		{name: "異なるOriginを拒否する", host: "reviews.example", origin: "https://evil.example", status: 403},
		{name: "cross-siteを拒否する", host: "reviews.example", fetchSite: "cross-site", status: 403},
		{name: "別Hostを拒否する", host: "evil.example", status: 403},
		{name: "suffixを拒否する", host: "reviews.example.evil.example", status: 403},
		{name: "別ポートを拒否する", host: "reviews.example:8443", status: 403},
		{name: "転送ヘッダーで迂回できない", host: "evil.example", forwarded: "reviews.example", status: 403},
		{name: "不正なHostを拒否する", host: "reviews.example:bad", status: 403},
		{name: "空Hostを拒否する", host: "", status: 403},
		{name: "指定した非既定ポートを許可する", publicURL: "https://reviews.example:8443/tools", host: "reviews.example:8443", status: 202},
		{name: "非既定ポートの省略を拒否する", publicURL: "https://reviews.example:8443/tools", host: "reviews.example", status: 403},
		{name: "IPv6表記を正規化する", publicURL: "https://[2001:db8::1]/tools", host: "[2001:0db8:0:0:0:0:0:1]:443", status: 202},
	} {
		t.Run(tc.name, func(t *testing.T) {
			publicURL := tc.publicURL
			if publicURL == "" {
				publicURL = "https://reviews.example/tools"
			}

			access, err := mcpaccess.New(publicURL)
			if err != nil {
				t.Fatal(err)
			}

			calls := 0
			handler, err := accesshandler.Protect(access, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.WriteHeader(202)
			}))
			if err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest("POST", "https://reviews.example/tools", nil)
			request.Host = tc.host
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}

			if tc.fetchSite != "" {
				request.Header.Set("Sec-Fetch-Site", tc.fetchSite)
			}

			if tc.forwarded != "" {
				request.Header.Set("Forwarded", "host="+tc.forwarded)
				request.Header.Set("X-Forwarded-Host", tc.forwarded)
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			expectedCalls := 0
			if tc.status == 202 {
				expectedCalls = 1
			}

			if response.Code != tc.status || calls != expectedCalls {
				t.Fatalf("status=%d calls=%d", response.Code, calls)
			}

			if response.Header().Get("WWW-Authenticate") != "" {
				t.Fatal("OAuthのchallengeを返した")
			}
		})
	}
}

func TestProtectConfiguration(t *testing.T) {
	valid, err := mcpaccess.New("https://reviews.example/tools")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name   string
		access *mcpaccess.Access
		next   http.Handler
	}{
		{"nil設定を拒否する", nil, http.NotFoundHandler()},
		{"ゼロ値を拒否する", &mcpaccess.Access{}, http.NotFoundHandler()},
		{"nilハンドラーを拒否する", valid, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, err := accesshandler.Protect(tc.access, tc.next)
			if handler != nil || !errors.Is(err, mcpaccess.ErrInvalidConfig) {
				t.Fatalf("handler=%v err=%v", handler, err)
			}
		})
	}
}
