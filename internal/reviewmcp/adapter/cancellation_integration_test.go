//go:build integration

package adapter

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestHTTPRequestCancellation(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"} {
		t.Run(version+"の切断が取得へ伝播する", func(t *testing.T) {
			started := make(chan struct{})
			finished := make(chan error, 1)
			handler, err := NewHandler(testutil.FetchFunc(func(ctx context.Context, _ steamdomain.AppID, _ steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				close(started)
				<-ctx.Done()
				finished <- ctx.Err()
				return steamdomain.ReviewPage{}, ctx.Err()
			}), mcpdomain.DefaultLimits(), localHTTPConfig(t))
			if err != nil {
				t.Fatal(err)
			}

			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			session := connectClient(t, server.URL, version)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			done := make(chan error, 1)
			go func() {
				_, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440}})
				done <- err
			}()
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("取得が開始しなかった")
			}

			cancel()
			select {
			case err := <-finished:
				if err != context.Canceled {
					t.Fatalf("取得終了=%v", err)
				}
			case <-time.After(2 * time.Second):
				t.Error("HTTP 切断が取得へ伝播しなかった")
			}

			select {
			case err := <-done:
				if err == nil {
					t.Error("キャンセルした呼び出しが成功した")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("クライアントが終了しなかった")
			}
		})
	}
}
