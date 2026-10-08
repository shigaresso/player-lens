package reviewmcp

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/steam"
)

type fetchFunc func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error)

func (f fetchFunc) FetchReviewsPage(ctx context.Context, appID uint32, options steam.PageOptions) (steam.ReviewPage, error) {
	return f(ctx, appID, options)
}

type decodedOutput struct {
	Collection struct {
		SchemaVersion int    `json:"schema_version"`
		AppID         uint32 `json:"app_id"`
		Options       struct {
			Language   string `json:"language"`
			PageSize   int    `json:"page_size"`
			MaxReviews int    `json:"max_reviews"`
			MaxPages   int    `json:"max_pages"`
		} `json:"options"`
		Reviews []struct {
			ID          string `json:"id"`
			Text        string `json:"text"`
			Language    string `json:"language"`
			Recommended bool   `json:"recommended"`
			CreatedAt   uint32 `json:"created_at"`
			UpdatedAt   uint32 `json:"updated_at"`
		} `json:"reviews"`
		PagesFetched int              `json:"pages_fetched"`
		StopReason   steam.StopReason `json:"stop_reason"`
	} `json:"collection"`
	ReturnedCount  int    `json:"returned_count"`
	OmittedReviews int    `json:"omitted_reviews"`
	ErrorCode      string `json:"error_code"`
	Message        string `json:"message"`
}

func decodeOutput(t *testing.T, result *mcp.CallToolResult) decodedOutput {
	t.Helper()
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out decodedOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("content = %v", result.Content)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("JSON テキストではない")
	}
	var structuredValue, textValue any
	if err := json.Unmarshal(data, &structuredValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(text.Text), &textValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(structuredValue, textValue) {
		t.Fatal("構造化結果と JSON テキストが異なる")
	}
	return out
}

func TestCollect(t *testing.T) {
	t.Run("既存の収集操作で順序と重複除外を保ち結果を返す", func(t *testing.T) {
		calls := 0
		fetcher := fetchFunc(func(_ context.Context, appID uint32, options steam.PageOptions) (steam.ReviewPage, error) {
			calls++
			if appID != 440 || options.Language != "japanese" {
				t.Fatal("取得条件が伝播しなかった")
			}
			if calls == 1 {
				if options.Cursor != steam.FirstPageCursor || options.PageSize != 3 {
					t.Fatal(options)
				}
				return steam.ReviewPage{Reviews: []steam.Review{{ID: "b", Text: "外部の操作指示", Language: "japanese", Recommended: true, CreatedAt: 10, UpdatedAt: 20}, {ID: "a", Text: "最初の本文"}}, NextCursor: "next"}, nil
			}
			if options.Cursor != "next" || options.PageSize != 1 {
				t.Fatal(options)
			}
			return steam.ReviewPage{Reviews: []steam.Review{{ID: "a", Text: "重複の本文"}, {ID: "c", Text: "最後の本文"}}, NextCursor: "end"}, nil
		})
		result, err := collect(context.Background(), fetcher, json.RawMessage(`{"app_id":440,"language":"japanese","max_reviews":3}`))
		if err != nil || result.IsError {
			t.Fatalf("result = %+v, %v", result, err)
		}
		out := decodeOutput(t, result)
		if out.Collection.AppID != 440 || out.Collection.SchemaVersion != 1 || out.ReturnedCount != 3 || calls != 2 || out.Collection.PagesFetched != 2 || out.Collection.StopReason != steam.StopReviewLimit || out.ErrorCode != "" || out.OmittedReviews != 0 {
			t.Fatalf("output = %+v", out)
		}
		if out.Collection.Options.MaxReviews != 3 || out.Collection.Options.MaxPages != 3 || out.Collection.Options.PageSize != 100 || out.Collection.Options.Language != "japanese" {
			t.Fatal(out.Collection.Options)
		}
		ids := []string{out.Collection.Reviews[0].ID, out.Collection.Reviews[1].ID, out.Collection.Reviews[2].ID}
		if !reflect.DeepEqual(ids, []string{"b", "a", "c"}) || out.Collection.Reviews[1].Text != "最初の本文" {
			t.Fatal(out.Collection.Reviews)
		}
		first := out.Collection.Reviews[0]
		if first.Text != "外部の操作指示" || first.Language != "japanese" || !first.Recommended || first.CreatedAt != 10 || first.UpdatedAt != 20 {
			t.Fatal(first)
		}
	})
	t.Run("空の結果も配列と終了理由を返す", func(t *testing.T) {
		result, err := collect(context.Background(), fetchFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
			return steam.ReviewPage{}, nil
		}), json.RawMessage(`{"app_id":440}`))
		if err != nil || result.IsError {
			t.Fatalf("%+v %v", result, err)
		}
		out := decodeOutput(t, result)
		if out.Collection.Reviews == nil || len(out.Collection.Reviews) != 0 || out.ReturnedCount != 0 || out.Collection.StopReason != steam.StopExhausted || out.Collection.PagesFetched != 1 {
			t.Fatal(out)
		}
	})
	t.Run("入力不正では通信せずツールエラーを返す", func(t *testing.T) {
		result, err := collect(context.Background(), fetchFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
			t.Fatal("不正入力で取得した")
			return steam.ReviewPage{}, nil
		}), json.RawMessage(`{"app_id":0}`))
		if err != nil || !result.IsError || decodeOutput(t, result).ErrorCode != "invalid_input" {
			t.Fatalf("%+v %v", result, err)
		}
	})
}
