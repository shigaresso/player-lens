//go:build integration

package adapter

import (
	"bytes"
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
				handler, err := NewHandler(testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
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
				}), mcpdomain.DefaultLimits(), localHTTPConfig(t), testutil.DiscardLogger(), time.Now)
				if err != nil {
					t.Fatal(err)
				}

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

func TestMCPInputFailurePrivacy(t *testing.T) {
	for _, version := range []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"} {
		t.Run(version+"で入力原因を公開せず利用枠を維持する", func(t *testing.T) {
			var calls atomic.Int32
			limits := testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 1, MaxConcurrent: 1})
			handler, err := NewHandler(testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				calls.Add(1)
				return steamdomain.ReviewPage{}, nil
			}), limits, localHTTPConfig(t), testutil.DiscardLogger(), time.Now)
			if err != nil {
				t.Fatal(err)
			}

			server := httptest.NewServer(handler)
			t.Cleanup(server.Close)
			session := connectClient(t, server.URL, version)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()

			var publicResponse []byte
			for _, tc := range []struct {
				name  string
				input string
			}{
				{"App ID が不正", `{"app_id":0}`},
				{"値の型が不正", `{"app_id":440,"page_size":"private_value"}`},
				{"未知の項目", `{"app_id":440,"private_field":"private_value"}`},
				{"項目が null", `{"app_id":440,"max_reviews":null}`},
				{"件数が上限超過", `{"app_id":440,"max_reviews":101}`},
				{"言語が不正", `{"app_id":440,"language":"private_value!"}`},
			} {
				t.Run(tc.name, func(t *testing.T) {
					result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "collect_reviews", Arguments: json.RawMessage(tc.input)})
					if err != nil || result == nil || !result.IsError {
						t.Fatalf("入力不正=%+v err=%v", result, err)
					}

					out := testutil.DecodeOutput(t, result)
					if out.ErrorCode != "invalid_input" || out.Message != "取得条件が不正です。app_id とツールの入力上限を確認してください。" || result.GetError() != nil || calls.Load() != 0 {
						t.Fatalf("公開情報か取得の契約を変更した: %+v calls=%d", out, calls.Load())
					}

					data, err := json.Marshal(result)
					if err != nil {
						t.Fatal(err)
					}

					if publicResponse == nil {
						publicResponse = data
					}

					if !bytes.Equal(data, publicResponse) {
						t.Fatal("入力の拒否理由により公開応答が変化した")
					}

					if strings.Contains(string(data), "private") || strings.Contains(string(data), "decode MCP") || strings.Contains(string(data), "must be") {
						t.Fatal("内部診断をクライアントへ返した")
					}
				})
			}

			result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 440}})
			if err != nil || result == nil || result.IsError || calls.Load() != 1 {
				t.Fatalf("不正入力が利用枠を消費した: result=%+v err=%v calls=%d", result, err, calls.Load())
			}
		})
	}
}
