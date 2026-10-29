//go:build integration

package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestMCPSharedUsageWindows(t *testing.T) {
	versions := []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"}
	for i, version := range versions {
		for _, tc := range []struct {
			name   string
			limits mcpdomain.Limits
			code   string
		}{
			{"分次の枠を共有する", testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 10, MaxConcurrent: 1}), "rate_limited"},
			{"日次の枠を共有する", testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 1, MaxConcurrent: 1}), "daily_limit_reached"},
		} {
			t.Run(version+"の別セッションと異なる版で"+tc.name, func(t *testing.T) {
				var calls atomic.Int32
				handler := NewHandler(testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
					calls.Add(1)
					return steamdomain.ReviewPage{}, nil
				}), tc.limits, localHTTPConfig(t))

				server := httptest.NewServer(handler)
				t.Cleanup(server.Close)
				first := connectClient(t, server.URL, version)
				second := connectClient(t, server.URL, versions[(i+1)%len(versions)])
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				if _, err := second.ListTools(ctx, nil); err != nil {
					t.Fatal(err)
				}

				invalid, err := second.CallTool(ctx, &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 0}})
				if err != nil || !invalid.IsError || testutil.DecodeOutput(t, invalid).ErrorCode != "invalid_input" {
					t.Fatalf("不正入力=%+v %v", invalid, err)
				}

				if _, err := second.CallTool(ctx, &mcp.CallToolParams{Name: "missing", Arguments: json.RawMessage(`{}`)}); err == nil {
					t.Fatal("未定義ツールが成功した")
				}

				params := &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440}}
				result, err := first.CallTool(ctx, params)
				if err != nil || result.IsError || calls.Load() != 1 {
					t.Fatalf("初期処理が枠を消費した: %+v %v", result, err)
				}

				result, err = second.CallTool(ctx, params)
				if err != nil {
					t.Fatal(err)
				}

				out := testutil.DecodeOutput(t, result)
				if !result.IsError || out.ErrorCode != tc.code || calls.Load() != 1 || out.Collection.PagesFetched != 0 || out.Collection.StopReason != steamdomain.StopFailed || out.ReturnedCount != 0 {
					t.Fatalf("共有制限=%+v calls=%d", out, calls.Load())
				}
			})
		}
	}
}

func TestMCPSharedConcurrentSlots(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"} {
		t.Run(version+"の実行中は別セッションを拒否し完了後に許可する", func(t *testing.T) {
			started := make(chan struct{})
			proceed := make(chan struct{})
			unblock := sync.OnceFunc(func() { close(proceed) })
			defer unblock()

			var calls atomic.Int32
			handler := NewHandler(testutil.FetchFunc(func(ctx context.Context, _ steamdomain.AppID, _ steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				if calls.Add(1) == 1 {
					close(started)
					select {
					case <-proceed:
					case <-ctx.Done():
						return steamdomain.ReviewPage{}, ctx.Err()
					}
				}

				return steamdomain.ReviewPage{}, nil
			}), testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 2, RequestsPerDay: 2, MaxConcurrent: 1}), localHTTPConfig(t))

			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			first := connectClient(t, server.URL, version)
			second := connectClient(t, server.URL, version)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			params := &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440}}
			done := make(chan error, 1)
			finished := make(chan struct{})
			defer func() {
				cancel()
				unblock()
				<-finished
			}()

			go func() {
				defer close(finished)

				result, err := first.CallTool(ctx, params)
				if err == nil && result.IsError {
					err = errors.New("受付済みの収集が失敗した")
				}

				done <- err
			}()
			select {
			case <-started:
			case <-ctx.Done():
				t.Fatal("取得が開始しなかった")
			}

			result, err := second.CallTool(ctx, params)
			if err != nil || !result.IsError || testutil.DecodeOutput(t, result).ErrorCode != "busy" || calls.Load() != 1 {
				t.Fatalf("実行中=%+v %v", result, err)
			}

			unblock()
			select {
			case err := <-done:
				if err != nil {
					t.Fatal(err)
				}
			case <-ctx.Done():
				t.Fatal("取得が完了しなかった")
			}

			result, err = second.CallTool(ctx, params)
			if err != nil || result.IsError || calls.Load() != 2 {
				t.Fatalf("返却後=%+v %v", result, err)
			}

			result, err = second.CallTool(ctx, params)
			if err != nil || !result.IsError || testutil.DecodeOutput(t, result).ErrorCode != "daily_limit_reached" || calls.Load() != 2 {
				t.Fatalf("開始回数=%+v %v", result, err)
			}
		})
	}
}
