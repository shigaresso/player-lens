//go:build integration

package adapter

import (
	"context"
	"encoding/json"
	"errors"
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

func TestMCPResultResponse(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25"} {
		for _, tc := range []struct {
			name    string
			code    string
			count   int
			omitted int
			pages   int
		}{
			{"途中失敗の部分結果をクライアントへ返す", "fetch_failed", 1, 0, 1},
			{"応答量超過をクライアントへ明示する", "response_too_large", 0, 1, 1},
			{"空の結果をクライアントへ返す", "", 0, 0, 1},
		} {
			t.Run(version+"で"+tc.name, func(t *testing.T) {
				var calls atomic.Int32
				handler := NewHandler(testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
					call := calls.Add(1)
					if tc.code == "response_too_large" {
						return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 10, Text: strings.Repeat("x", 70000)}}, NextCursor: "next"}, nil
					}

					if tc.code == "fetch_failed" {
						if call == 1 {
							return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 11, Text: "取得済み"}}, NextCursor: "next"}, nil
						}

						return steamdomain.ReviewPage{}, errors.New("secret internal error")
					}

					return steamdomain.ReviewPage{}, nil
				}), mcpdomain.DefaultLimits(), localHTTPConfig(t))

				server := httptest.NewServer(handler)
				t.Cleanup(server.Close)
				session := connectClient(t, server.URL, version)
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				count := 3
				if tc.code == "response_too_large" {
					count = 1
				}

				result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440, "max_reviews": count}})
				if err != nil || result.IsError != (tc.code != "") {
					t.Fatalf("%+v %v", result, err)
				}

				out := testutil.DecodeOutput(t, result)
				if out.ErrorCode != tc.code || out.ReturnedCount != tc.count || out.OmittedReviews != tc.omitted || out.Collection.PagesFetched != tc.pages || out.Collection.Reviews == nil {
					t.Fatal(out)
				}

				data, err := json.Marshal(result)
				if err != nil {
					t.Fatal(err)
				}

				if len(data) > 128*1024 || strings.Contains(string(data), "secret") {
					t.Fatal("上限または内部情報の契約が失われた")
				}

				if tc.count == 1 && out.Collection.Reviews[0].RecommendationID != "11" {
					t.Fatal(out)
				}
			})
		}
	}
}
