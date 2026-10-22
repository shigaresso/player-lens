//go:build integration

package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp"
	"github.com/shigaresso/player-lens/internal/steam"
)

type requestFetcher struct {
	started chan struct{}
	release chan struct{}
}

func (f requestFetcher) FetchReviewsPage(ctx context.Context, _ uint32, _ steam.PageOptions) (steam.ReviewPage, error) {
	close(f.started)
	select {
	case <-f.release:
		return steam.ReviewPage{Reviews: []steam.Review{{ID: "finish", Text: "取得完了"}}, NextCursor: "next"}, nil
	case <-ctx.Done():
		return steam.ReviewPage{}, ctx.Err()
	}
}

func TestMCPShutdownWaitsForActiveCollection(t *testing.T) {
	fetcher := requestFetcher{started: make(chan struct{}), release: make(chan struct{})}
	release := sync.OnceFunc(func() { close(fetcher.release) })
	t.Cleanup(release)
	mcpHandler, err := reviewmcp.NewHandler(fetcher, reviewmcp.DefaultLimits(), nil)
	if err != nil {
		t.Fatal(err)
	}
	url, httpClient, cancel, done, closed := startTestServer(t, newHandler(mcpHandler), 2*time.Second)
	client := mcp.NewClient(&mcp.Implementation{Name: "server-integration", Version: "1"}, nil)
	ctx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: url + "/tools", HTTPClient: httpClient}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})
	resultDone := make(chan struct{})
	var result *mcp.CallToolResult
	var callErr error
	go func() {
		result, callErr = session.CallTool(ctx, &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440, "max_reviews": 1}})
		close(resultDone)
	}()
	t.Cleanup(func() {
		release()
		stop()
		select {
		case <-resultDone:
		case <-time.After(2 * time.Second):
			t.Error("MCP 呼び出しの goroutine が終了しなかった")
		}
	})
	select {
	case <-fetcher.started:
	case <-ctx.Done():
		t.Fatal("MCP 取得が開始しなかった")
	}
	cancel()
	select {
	case <-closed:
	case <-ctx.Done():
		t.Fatal("新規受付を停止しなかった")
	}
	release()
	select {
	case <-resultDone:
	case <-ctx.Done():
		t.Fatal("処理中の MCP 応答が返らなかった")
	}
	if callErr != nil || result.IsError {
		t.Fatalf("result=%+v err=%v", result, callErr)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("サーバーが終了しなかった")
	}
	if response, err := httpClient.Get(url + "/healthz"); err == nil {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
		t.Fatal("終了後に新規受付した")
	}
}
