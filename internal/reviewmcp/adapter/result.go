package adapter

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	jsonoutput "github.com/shigaresso/player-lens/internal/jsonoutput/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type toolOutput struct {
	Collection     json.RawMessage `json:"collection"`
	ReturnedCount  int             `json:"returned_count"`
	OmittedReviews int             `json:"omitted_reviews"`
	ErrorCode      string          `json:"error_code"`
	Message        string          `json:"message"`
	AnalysisGuide  analysisGuide   `json:"analysis_guide"`
}

const maxResultBytes = 128 * 1024

var errResponseTooLarge = errors.New("MCP collection response exceeds size limit")

// toolResultEncodingError は公開する文言と内部の原因を分けた応答生成エラー。
type toolResultEncodingError struct {
	cause error
}

// Error は内部の原因を含まない固定の説明を返す。
func (e *toolResultEncodingError) Error() string {
	return "MCP の収集結果を生成できませんでした。"
}

// Unwrap は変換と収集の原因を errors.Is・errors.As で追跡できる形で返す。
func (e *toolResultEncodingError) Unwrap() error {
	return e.cause
}

// encodeCollectionResult は公開用の失敗情報へ変換し、内部の原因を結果または戻りエラーへ保持する。
func encodeCollectionResult(collection steamdomain.Collection, collectErr error) (*mcp.CallToolResult, error) {
	failure := collectionFailure(collectErr)
	result, err := encodeResult(collection, failure)
	if err != nil {
		return nil, &toolResultEncodingError{cause: errors.Join(err, collectErr)}
	}

	if collectErr != nil {
		result.SetError(errors.Join(result.GetError(), collectErr))
	}

	return result, nil
}

// encodeResult は通常応答のサイズを判定し、超過時は本文を一度だけ省略する。
// 省略後も上限を超える場合は結果を返さず、errResponseTooLarge を保持したエラーを返す。
func encodeResult(collection steamdomain.Collection, failure toolFailureInfo) (*mcp.CallToolResult, error) {
	result, err := marshalToolResult(collection, failure, 0)
	if err != nil {
		return nil, err
	}

	withinLimit, err := toolResultWithinLimit(result)
	if err != nil {
		return nil, err
	}

	if withinLimit {
		return result, nil
	}

	result, err = encodeOversizedResult(collection)
	if err != nil {
		return nil, err
	}

	withinLimit, err = toolResultWithinLimit(result)
	if err != nil {
		return nil, err
	}

	if !withinLimit {
		return nil, fmt.Errorf("encode MCP result after omitting reviews: %w", errResponseTooLarge)
	}

	return result, nil
}

// toolResultWithinLimit は SDK の結果全体を JSON 化し、応答量の上限内かを返す。
func toolResultWithinLimit(result *mcp.CallToolResult) (bool, error) {
	serialized, err := json.Marshal(result)
	if err != nil {
		return false, fmt.Errorf("measure MCP result: %w", err)
	}

	return len(serialized) <= maxResultBytes, nil
}

// encodeOversizedResult は本文を除き、省略件数と公開情報を設定した応答を生成する。
// サイズの判定と生成手順は encodeResult が担当する。
func encodeOversizedResult(collection steamdomain.Collection) (*mcp.CallToolResult, error) {
	count := len(collection.Reviews)
	collection.Reviews = nil
	failure := toolFailureInfo{
		code:    "response_too_large",
		message: "応答量の上限を超えたためレビュー本文を返せません。max_reviews を減らしてください。",
	}
	result, err := marshalToolResult(collection, failure, count)
	if err != nil {
		return nil, err
	}

	result.SetError(errResponseTooLarge)
	return result, nil
}

// marshalToolResult は同じ JSON を構造化結果とテキストへ設定し、入力を変更しない。
func marshalToolResult(collection steamdomain.Collection, failure toolFailureInfo, omitted int) (*mcp.CallToolResult, error) {
	data, err := jsonoutput.MarshalCollection(collection)
	if err != nil {
		return nil, fmt.Errorf("convert MCP collection: %w", err)
	}

	out := toolOutput{
		Collection: data, ReturnedCount: len(collection.Reviews), OmittedReviews: omitted,
		ErrorCode: failure.code, Message: failure.message,
		AnalysisGuide: analysisGuide{Version: analysisGuideVersion, Instructions: analysisGuideInstructions},
	}
	content, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("encode MCP result: %w", err)
	}

	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(content)}},
		StructuredContent: json.RawMessage(content),
		IsError:           failure.code != "",
	}, nil
}
