//go:build integration

package adapter

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestMCPReviewType(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"} {
		t.Run(version+"で評価種別と非消費と拒否結果を確認する", func(t *testing.T) {
			var calls atomic.Int32
			handler, err := NewHandler(fetchFunc(func(_ context.Context, id steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				calls.Add(1)
				if id != 440 {
					t.Errorf("AppID=%d", id)
				}
				return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 1, Recommended: options.ReviewType == steamdomain.ReviewTypePositive}}, NextCursor: "next"}, nil
			}), testLimits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 3, RequestsPerDay: 10, MaxConcurrent: 1}), nil)
			if err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			session := connectClient(t, server.URL, version)
			tools, err := session.ListTools(t.Context(), nil)
			if err != nil || len(tools.Tools) != 1 {
				t.Fatalf("tools=%+v err=%v", tools, err)
			}
			schemaData, err := json.Marshal(tools.Tools[0].InputSchema)
			if err != nil {
				t.Fatal(err)
			}
			var schema struct {
				Properties map[string]struct {
					Enum []string `json:"enum"`
				} `json:"properties"`
			}
			if err := json.Unmarshal(schemaData, &schema); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(schema.Properties["review_type"].Enum, []string{"", "all", "positive", "negative"}) {
				t.Fatalf("schema=%s", schemaData)
			}
			for _, bad := range []struct {
				name  string
				value any
			}{
				{"未知値", "mixed"}, {"null", nil}, {"数値", 1}, {"前後空白", " negative "}, {"大文字", "Positive"},
			} {
				t.Run(bad.name+"は通信と利用枠を消費しない", func(t *testing.T) {
					result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440, "review_type": bad.value}})
					if err != nil || result == nil {
						t.Fatalf("result=%+v err=%v", result, err)
					}
					if !result.IsError || decodeOutput(t, result).ErrorCode != "invalid_input" || calls.Load() != 0 {
						t.Fatalf("result=%+v calls=%d", result, calls.Load())
					}
				})
			}
			for i, kind := range []string{"all", "positive", "negative"} {
				t.Run(kind+"を収集して条件を返す", func(t *testing.T) {
					result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440, "review_type": kind, "max_reviews": 1, "max_pages": 1}})
					if err != nil || result == nil {
						t.Fatalf("result=%+v err=%v", result, err)
					}
					out := decodeOutput(t, result)
					if result.IsError || out.Collection.Options.ReviewType != kind || out.ReturnedCount != 1 || calls.Load() != int32(i+1) || out.Collection.Reviews[0].Recommended != (kind == "positive") {
						t.Fatalf("out=%+v calls=%d", out, calls.Load())
					}
				})
			}
			result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440, "review_type": "negative"}})
			if err != nil || result == nil {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			out := decodeOutput(t, result)
			if !result.IsError || out.ErrorCode != "rate_limited" || out.Collection.Options.ReviewType != "negative" || out.ReturnedCount != 0 || out.Collection.PagesFetched != 0 || calls.Load() != 3 {
				t.Fatalf("out=%+v calls=%d", out, calls.Load())
			}
		})
	}
}
