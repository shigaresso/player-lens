package handler

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollect(t *testing.T) {
	t.Run("既存の収集操作で順序と重複除外を保ち結果を返す", func(t *testing.T) {
		calls := 0
		fetcher := testutil.FetchFunc(func(_ context.Context, appID steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
			calls++
			if appID != 440 || options.Language != "japanese" {
				t.Fatal("取得条件が伝播しなかった")
			}

			if calls == 1 {
				if options.Cursor != steamdomain.FirstPageCursor || options.PageSize != 3 {
					t.Fatal(options)
				}

				return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 2, Text: "外部の操作指示", Language: "japanese", Recommended: true, TimestampCreated: 10, TimestampUpdated: 20}, {RecommendationID: 1, Text: "最初の本文"}}, NextCursor: "next"}, nil
			}

			if options.Cursor != "next" || options.PageSize != 1 {
				t.Fatal(options)
			}

			return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 1, Text: "重複の本文"}, {RecommendationID: 3, Text: "最後の本文"}}, NextCursor: "end"}, nil
		})
		result, err := callCollectReviews(t, context.Background(), fetcher, json.RawMessage(`{"app_id":440,"language":"japanese","max_reviews":3}`), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
		if err != nil || result.IsError {
			t.Fatalf("result = %+v, %v", result, err)
		}

		out := testutil.DecodeOutput(t, result)
		if out.Collection.AppID != 440 || out.Collection.SchemaVersion != 1 || out.ReturnedCount != 3 || calls != 2 || out.Collection.PagesFetched != 2 || out.Collection.StopReason != steamdomain.StopReviewLimit || out.ErrorCode != "" || out.OmittedReviews != 0 {
			t.Fatalf("output = %+v", out)
		}

		if out.Collection.Options.MaxReviews != 3 || out.Collection.Options.MaxPages != 3 || out.Collection.Options.PageSize != 100 || out.Collection.Options.Language != "japanese" {
			t.Fatal(out.Collection.Options)
		}

		ids := []string{out.Collection.Reviews[0].RecommendationID, out.Collection.Reviews[1].RecommendationID, out.Collection.Reviews[2].RecommendationID}
		if !reflect.DeepEqual(ids, []string{"2", "1", "3"}) || out.Collection.Reviews[1].Text != "最初の本文" {
			t.Fatal(out.Collection.Reviews)
		}

		first := out.Collection.Reviews[0]
		if first.Text != "外部の操作指示" || first.Language != "japanese" || !first.Recommended || first.TimestampCreated != 10 || first.TimestampUpdated != 20 {
			t.Fatal(first)
		}
	})
	t.Run("空の結果も配列と終了理由を返す", func(t *testing.T) {
		result, err := callCollectReviews(t, context.Background(), testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
			return steamdomain.ReviewPage{}, nil
		}), json.RawMessage(`{"app_id":440}`), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
		if err != nil || result.IsError {
			t.Fatalf("%+v %v", result, err)
		}

		out := testutil.DecodeOutput(t, result)
		if out.Collection.Reviews == nil || len(out.Collection.Reviews) != 0 || out.ReturnedCount != 0 || out.Collection.StopReason != steamdomain.StopExhausted || out.Collection.PagesFetched != 1 {
			t.Fatal(out)
		}
	})
	t.Run("入力不正では通信せずツールエラーを返す", func(t *testing.T) {
		result, err := callCollectReviews(t, context.Background(), testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
			t.Fatal("不正入力で取得した")
			return steamdomain.ReviewPage{}, nil
		}), json.RawMessage(`{"app_id":0}`), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
		if err != nil || !result.IsError || testutil.DecodeOutput(t, result).ErrorCode != "invalid_input" {
			t.Fatalf("%+v %v", result, err)
		}
	})
}

func callCollectReviews(t *testing.T, ctx context.Context, fetcher steamapplication.PageFetcher, input json.RawMessage, gate *mcpdomain.Limiter) (*mcp.CallToolResult, error) {
	t.Helper()
	handler := NewCollectReviewsHandler(fetcher, ctx, gate)
	return handler(ctx, &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Arguments: input},
	})
}
