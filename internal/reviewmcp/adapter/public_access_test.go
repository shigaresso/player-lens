package adapter

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestPublicMCPAccess(t *testing.T) {
	access, err := mcpaccess.New("https://reviews.example/tools")
	if err != nil {
		t.Fatal(err)
	}

	handler, err := NewHandler(fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		t.Fatal("受付検査でSteamに接続した")
		return steamdomain.ReviewPage{}, nil
	}), mcpdomain.DefaultLimits(), access)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, host, origin, body string
		status                   int
	}{
		{"認証なしで公開Hostから一覧を取得できる", "reviews.example", "", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, 200},
		{"不許可Hostを拒否する", "evil.example", "", `{}`, 403},
		{"別Originを拒否する", "reviews.example", "https://evil.example", `{}`, 403},
		{"公開時も本文上限を維持する", "reviews.example", "", strings.Repeat("x", 16*1024+1), 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "https://reviews.example/tools", strings.NewReader(tc.body))
			request.Host = tc.host
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "application/json, text/event-stream")
			request.Header.Set("MCP-Protocol-Version", "2025-11-25")
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body)
			}

			if response.Header().Get("WWW-Authenticate") != "" {
				t.Fatal("OAuthのchallengeを返した")
			}

			if tc.status == 200 && !strings.Contains(response.Body.String(), "collect_reviews") {
				t.Fatal(response.Body.String())
			}
		})
	}
}
