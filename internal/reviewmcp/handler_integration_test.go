//go:build integration

package reviewmcp

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/steam"
)

func connectClient(t *testing.T, handlerURL, version string) *mcp.ClientSession {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	client := mcp.NewClient(&mcp.Implementation{Name: "player-lens-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: handlerURL}, &mcp.ClientSessionOptions{ProtocolVersion: version})
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

func TestMCPProtocol(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"} {
		t.Run(version+"で初期処理とツール呼び出しができる", func(t *testing.T) {
			var calls atomic.Int32
			handler, err := NewHandler(fetchFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
				calls.Add(1)
				return steam.ReviewPage{Reviews: []steam.Review{{ID: "sdk", Text: "分析用の本文"}}, NextCursor: "next"}, nil
			}), DefaultLimits())
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			session := connectClient(t, server.URL, version)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			tools, err := session.ListTools(ctx, nil)
			if err != nil || len(tools.Tools) != 1 || tools.Tools[0].Name != "collect_reviews" {
				t.Fatalf("tools=%+v err=%v", tools, err)
			}
			tool := tools.Tools[0]
			if tool.InputSchema == nil || tool.OutputSchema == nil || tool.Annotations == nil || !tool.Annotations.ReadOnlyHint || tool.Annotations.OpenWorldHint == nil || !*tool.Annotations.OpenWorldHint {
				t.Fatal("ツール契約が公開されていない")
			}
			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440, "max_reviews": 1}})
			if err != nil || result.IsError {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			out := decodeOutput(t, result)
			if calls.Load() != 1 || out.ReturnedCount != 1 || out.Collection.Reviews[0].ID != "sdk" || out.Collection.StopReason != steam.StopReviewLimit {
				t.Fatal(out)
			}
			invalid, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 0}})
			if err != nil || !invalid.IsError || decodeOutput(t, invalid).ErrorCode != "invalid_input" || calls.Load() != 1 {
				t.Fatalf("入力エラー=%+v %v calls=%d", invalid, err, calls.Load())
			}
			_, err = session.CallTool(ctx, &mcp.CallToolParams{Name: "missing", Arguments: json.RawMessage(`{}`)})
			if err == nil || calls.Load() != 1 {
				t.Fatal("未定義のツールをプロトコルエラーとしなかった")
			}
		})
	}
}
