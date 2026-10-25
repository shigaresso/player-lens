//go:build integration

package adapter

import (
	"context"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestMCPSlotDuringResponseConversion(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"} {
		t.Run(version+"で応答変換中は別セッションを拒否し完了後に受付する", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			converting := make(chan struct{})
			proceed := make(chan struct{})
			unblock := sync.OnceFunc(func() { close(proceed) })
			defer unblock()
			cause := &conversionProbeError{classify: sync.OnceFunc(func() {
				close(converting)
				select {
				case <-proceed:
				case <-ctx.Done():
				}
			})}
			var calls atomic.Int32
			handler, err := NewHandler(fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				if calls.Add(1) == 1 {
					return steamdomain.ReviewPage{}, cause
				}
				return steamdomain.ReviewPage{}, nil
			}), testLimits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 10, MaxConcurrent: 1}), nil)
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			first := connectClient(t, server.URL, version)
			second := connectClient(t, server.URL, version)
			params := &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440}}
			type response struct {
				result *mcp.CallToolResult
				err    error
			}
			done := make(chan response, 1)
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				result, err := first.CallTool(ctx, params)
				done <- response{result, err}
			}()
			t.Cleanup(func() {
				unblock()
				cancel()
				<-finished
			})
			select {
			case <-converting:
			case <-ctx.Done():
				t.Fatal("応答変換が始まらなかった", ctx.Err())
			}
			busy, err := second.CallTool(ctx, params)
			if err != nil || busy == nil || !busy.IsError || decodeOutput(t, busy).ErrorCode != "busy" || calls.Load() != 1 {
				t.Fatalf("応答変換中の別セッション=%+v err=%v calls=%d", busy, err, calls.Load())
			}
			unblock()
			select {
			case firstResponse := <-done:
				if firstResponse.err != nil || firstResponse.result == nil || !firstResponse.result.IsError || decodeOutput(t, firstResponse.result).ErrorCode != "fetch_failed" {
					t.Fatalf("最初の応答=%+v", firstResponse)
				}
			case <-ctx.Done():
				t.Fatal("応答変換が完了しなかった", ctx.Err())
			}
			result, err := second.CallTool(ctx, params)
			if err != nil || result == nil || result.IsError || calls.Load() != 2 {
				t.Fatalf("応答変換後の再受付=%+v err=%v calls=%d", result, err, calls.Load())
			}
		})
	}
}
