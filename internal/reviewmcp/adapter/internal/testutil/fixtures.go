// Package testutil は MCP の契約テストで使う fixture とアサーションを共有する。
package testutil

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// FetchFunc はテストごとの取得結果を注入する関数型である。
//
//contractcheck:data テスト用の取得処理を差し替える関数データ
type FetchFunc func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error)

// FetchReviewsPage は注入した取得関数を呼ぶ。
func (f FetchFunc) FetchReviewsPage(ctx context.Context, appID steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
	return f(ctx, appID, options)
}

// DecodedOutput は公開する MCP 結果を検査するための受信型である。
//
//contractcheck:data テストで MCP の応答を読み取る受信用データ
type DecodedOutput struct {
	Collection struct {
		SchemaVersion int               `json:"schema_version"`
		AppID         steamdomain.AppID `json:"app_id"`
		Options       struct {
			ReviewType string `json:"review_type"`
			Language   string `json:"language"`
			PageSize   int    `json:"page_size"`
			MaxReviews int    `json:"max_reviews"`
			MaxPages   int    `json:"max_pages"`
		} `json:"options"`
		Reviews []struct {
			RecommendationID string                       `json:"id"`
			Text             string                       `json:"text"`
			Language         string                       `json:"language"`
			Recommended      bool                         `json:"recommended"`
			TimestampCreated steamdomain.TimestampCreated `json:"created_at"`
			TimestampUpdated steamdomain.TimestampUpdated `json:"updated_at"`
		} `json:"reviews"`
		PagesFetched int                    `json:"pages_fetched"`
		StopReason   steamdomain.StopReason `json:"stop_reason"`
	} `json:"collection"`
	ReturnedCount  int    `json:"returned_count"`
	OmittedReviews int    `json:"omitted_reviews"`
	ErrorCode      string `json:"error_code"`
	Message        string `json:"message"`
	AnalysisGuide  struct {
		Version      string `json:"version"`
		Instructions string `json:"instructions"`
	} `json:"analysis_guide"`
}

// DecodeOutput は構造化結果とテキスト・ガイドの契約を検査して結果を返す。
func DecodeOutput(t *testing.T, result *mcp.CallToolResult) DecodedOutput {
	t.Helper()
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}

	var out DecodedOutput
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}

	if out.AnalysisGuide.Version != "1" || out.AnalysisGuide.Instructions == "" {
		t.Fatal("版1の分析ガイドがない")
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

// Limits はテストで使う有効な利用上限を生成する。
func Limits(t *testing.T, config mcpdomain.LimitsConfig) mcpdomain.Limits {
	t.Helper()
	limits, err := mcpdomain.NewLimits(config)
	if err != nil {
		t.Fatal(err)
	}

	return limits
}

// ConversionProbeError は公開用の失敗分類が原因を調べる時点を観測する。
// Error は収集時のラップでも呼ばれるため、errors.Is による分類を使う。
//
//contractcheck:data テストでエラー分類の呼び出しを観測するデータ
type ConversionProbeError struct{ Classify func() }

// Error は取得失敗の固定表現を返す。
func (*ConversionProbeError) Error() string { return "fetch failed" }

// Is は失敗分類の開始時に Classify を同期的に呼ぶ。
func (e *ConversionProbeError) Is(target error) bool {
	if target == context.Canceled {
		e.Classify()
	}

	return false
}
