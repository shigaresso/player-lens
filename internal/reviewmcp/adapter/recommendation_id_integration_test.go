//go:build integration

package adapter

import (
	"context"
	"io"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestMCPNumericRecommendationIDs(t *testing.T) {
	for _, tc := range []struct {
		name       string
		partial    bool
		maxReviews int
		wantCode   string
		wantReason steamdomain.StopReason
		wantCalls  int32
	}{
		{"成功時にゼロと最大IDを文字列で返す", false, 2, "", steamdomain.StopReviewLimit, 1},
		{"途中失敗でもゼロと最大IDを文字列で返す", true, 3, "fetch_failed", steamdomain.StopFailed, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			handler := NewHandler(testutil.FetchFunc(func(_ context.Context, appID steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				if appID != 4294967295 {
					t.Errorf("appID=%d", appID)
				}

				if calls.Add(1) > 1 {
					return steamdomain.ReviewPage{}, io.ErrUnexpectedEOF
				}

				return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 0}, {RecommendationID: 18446744073709551615}}, NextCursor: "next"}, nil
			}), mcpdomain.DefaultLimits(), localHTTPConfig(t))

			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			session := connectClient(t, server.URL, "2026-07-28")
			result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": uint32(4294967295), "max_reviews": tc.maxReviews}})
			if err != nil || result.IsError != tc.partial {
				t.Fatalf("result=%+v error=%v", result, err)
			}

			out := testutil.DecodeOutput(t, result)
			if out.Collection.AppID != 4294967295 || out.ErrorCode != tc.wantCode || out.Collection.StopReason != tc.wantReason || out.ReturnedCount != 2 || len(out.Collection.Reviews) != 2 || calls.Load() != tc.wantCalls {
				t.Fatalf("output=%+v calls=%d", out, calls.Load())
			}

			if ids := []string{out.Collection.Reviews[0].RecommendationID, out.Collection.Reviews[1].RecommendationID}; !reflect.DeepEqual(ids, []string{"0", "18446744073709551615"}) {
				t.Fatalf("ID=%v", ids)
			}
		})
	}
}
