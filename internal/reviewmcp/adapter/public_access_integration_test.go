//go:build integration

package adapter

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type publicHostTransport struct{ base http.RoundTripper }

func (p publicHostTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.Header.Get("Authorization") != "" {
		return nil, errors.New("匿名接続に認証情報が含まれた")
	}

	request := r.Clone(r.Context())
	request.Host = "reviews.example"
	return p.base.RoundTrip(request)
}

func TestPublicMCPProtocolsAndSharedBudget(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"} {
		t.Run(version+"で匿名接続し別セッションと利用枠を共有する", func(t *testing.T) {
			config, err := NewHTTPConfig("https://reviews.example/tools")
			if err != nil {
				t.Fatal(err)
			}

			var calls atomic.Int32
			handler := NewHandler(testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				calls.Add(1)
				return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 16, Text: "分析用の本文"}}, NextCursor: "next"}, nil
			}), testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 10, MaxConcurrent: 1}), config)

			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			for _, rejected := range []struct{ name, host, origin string }{
				{"不許可Host", "evil.example", ""},
				{"不許可Origin", "reviews.example", "https://evil.example"},
			} {
				t.Run(rejected.name+"は収集枠を消費しない", func(t *testing.T) {
					request, err := http.NewRequestWithContext(ctx, "POST", server.URL+"/tools", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"collect_reviews","arguments":{"app_id":440}}}`))
					if err != nil {
						t.Fatal(err)
					}

					request.Host = rejected.host
					request.Header.Set("Content-Type", "application/json")
					request.Header.Set("Accept", "application/json, text/event-stream")
					if rejected.origin != "" {
						request.Header.Set("Origin", rejected.origin)
					}

					response, err := server.Client().Do(request)
					if err != nil {
						t.Fatal(err)
					}

					if err := response.Body.Close(); err != nil {
						t.Fatal(err)
					}

					if response.StatusCode != 403 || calls.Load() != 0 {
						t.Fatalf("status=%d calls=%d", response.StatusCode, calls.Load())
					}
				})
			}

			connect := func() *mcp.ClientSession {
				t.Helper()
				client := mcp.NewClient(&mcp.Implementation{Name: "public-test", Version: "1"}, nil)
				session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/tools", HTTPClient: &http.Client{Transport: publicHostTransport{base: server.Client().Transport}}}, &mcp.ClientSessionOptions{ProtocolVersion: version})
				if err != nil {
					t.Fatal(err)
				}

				t.Cleanup(func() {
					if err := session.Close(); err != nil {
						t.Error(err)
					}
				})
				return session
			}
			first, second := connect(), connect()
			tools, err := second.ListTools(ctx, nil)
			if err != nil || len(tools.Tools) != 1 || tools.Tools[0].Name != "collect_reviews" {
				t.Fatalf("tools=%+v err=%v", tools, err)
			}

			invalid, err := second.CallTool(ctx, &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 0}})
			if err != nil || !invalid.IsError || testutil.DecodeOutput(t, invalid).ErrorCode != "invalid_input" {
				t.Fatalf("invalid=%+v err=%v", invalid, err)
			}

			params := &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440, "max_reviews": 1}}
			result, err := first.CallTool(ctx, params)
			if err != nil || result.IsError {
				t.Fatalf("result=%+v err=%v", result, err)
			}

			out := testutil.DecodeOutput(t, result)
			if out.ReturnedCount != 1 || out.Collection.Reviews[0].RecommendationID != "16" || calls.Load() != 1 {
				t.Fatalf("out=%+v calls=%d", out, calls.Load())
			}

			result, err = second.CallTool(ctx, params)
			if err != nil || !result.IsError || testutil.DecodeOutput(t, result).ErrorCode != "rate_limited" || calls.Load() != 1 {
				t.Fatalf("result=%+v err=%v calls=%d", result, err, calls.Load())
			}
		})
	}
}
