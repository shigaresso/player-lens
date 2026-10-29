package adapter_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	accessadapter "github.com/shigaresso/player-lens/internal/mcpaccess/adapter"
	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

func TestProtectLocal(t *testing.T) {
	for _, tc := range []struct {
		name, host, origin, forwarded, forwardedHost, fetchSite string
		status                                                  int
		hostRejected                                            bool
	}{
		{name: "ポートなしのlocalhostを許可する", host: "localhost", status: 202},
		{name: "大文字のlocalhostとポートを許可する", host: "LOCALHOST:8080", status: 202},
		{name: "IPv4ループバックを許可する", host: "127.0.0.1:8080", status: 202},
		{name: "127の別のループバックIPも許可する", host: "127.0.0.2:8080", status: 202},
		{name: "IPv6ループバックを許可する", host: "[::1]:8080", status: 202},
		{name: "省略していないIPv6ループバックを許可する", host: "[0:0:0:0:0:0:0:1]:8080", status: 202},
		{name: "一致するOriginを許可する", host: "localhost:8080", origin: "http://localhost:8080", status: 202},
		{name: "空Hostを拒否する", host: "", status: 403, hostRejected: true},
		{name: "外部Hostを拒否する", host: "external.example:8080", status: 403, hostRejected: true},
		{name: "localhostに似た外部Hostを拒否する", host: "localhost.external.example", status: 403, hostRejected: true},
		{name: "コンテナのIPを拒否する", host: "172.17.0.2:8080", status: 403, hostRejected: true},
		{name: "外部IPv6を拒否する", host: "[2001:db8::1]:8080", status: 403, hostRejected: true},
		{name: "ForwardedでHostを偽装しても拒否する", host: "external.example", forwarded: "host=localhost", status: 403, hostRejected: true},
		{name: "XForwardedHostでHostを偽装しても拒否する", host: "external.example", forwardedHost: "localhost", status: 403, hostRejected: true},
		{name: "許可Hostでも異なるOriginを拒否する", host: "localhost:8080", origin: "https://external.example", status: 403},
		{name: "許可Hostでもcross-siteを拒否する", host: "localhost:8080", fetchSite: "cross-site", status: 403},
		{name: "HostとOriginの両方が不許可ならHostの説明を返す", host: "external.example", origin: "https://other.example", status: 403, hostRejected: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			access := mcpaccess.NewLocal()

			calls := 0
			handler, err := accessadapter.Protect(access, "ローカルの Host のみ利用できます", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.Header().Set("X-Next", "reached")
				w.WriteHeader(http.StatusAccepted)
				if _, err := io.WriteString(w, "accepted\n"); err != nil {
					t.Errorf("後続ハンドラーの応答を書き込めなかった: %v", err)
				}
			}))
			if err != nil {
				t.Fatal(err)
			}

			request := httptest.NewRequest(http.MethodPost, "http://localhost:8080/tools", nil)
			request.Host = tc.host
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}

			if tc.fetchSite != "" {
				request.Header.Set("Sec-Fetch-Site", tc.fetchSite)
			}

			if tc.forwarded != "" {
				request.Header.Set("Forwarded", tc.forwarded)
			}

			if tc.forwardedHost != "" {
				request.Header.Set("X-Forwarded-Host", tc.forwardedHost)
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			wantCalls := 0
			if tc.status == http.StatusAccepted {
				wantCalls = 1
			}

			if response.Code != tc.status || calls != wantCalls {
				t.Fatalf("status=%d calls=%d; want status=%d calls=%d", response.Code, calls, tc.status, wantCalls)
			}

			if wantCalls == 1 && (response.Header().Get("X-Next") != "reached" || response.Body.String() != "accepted\n") {
				t.Fatalf("後続ハンドラーの応答が変わった: header=%v body=%q", response.Header(), response.Body.String())
			}

			if tc.hostRejected && response.Body.String() != "ローカルの Host のみ利用できます\n" {
				t.Fatalf("Hostの拒否理由が変わった: %q", response.Body.String())
			}
		})
	}
}
