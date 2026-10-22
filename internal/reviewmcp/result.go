package reviewmcp

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/jsonoutput"
	"github.com/shigaresso/player-lens/internal/steam"
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

// encodeCollectionResult は公開用の失敗情報へ変換し、内部の原因を結果へ保持する。
func encodeCollectionResult(collection steam.Collection, collectErr error) (*mcp.CallToolResult, error) {
	failure := collectionFailure(collectErr)
	result, err := encodeResult(collection, failure.code, failure.message, 0)
	if err != nil {
		return nil, err
	}
	if collectErr != nil {
		result.SetError(errors.Join(result.GetError(), collectErr))
	}
	return result, nil
}

// encodeResult は SDK の結果全体の JSON サイズを計測し、超過時は本文を省略する。
func encodeResult(collection steam.Collection, code, message string, omitted int) (*mcp.CallToolResult, error) {
	result, err := marshalToolResult(collection, code, message, omitted)
	if err != nil {
		return nil, err
	}
	serialized, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("measure MCP result: %w", err)
	}
	if len(serialized) <= maxResultBytes {
		return result, nil
	}
	return encodeOversizedResult(collection)
}

func encodeOversizedResult(collection steam.Collection) (*mcp.CallToolResult, error) {
	count := len(collection.Reviews)
	collection.Reviews = nil
	result, err := encodeResult(collection, "response_too_large", "応答量の上限を超えたためレビュー本文を返せません。max_reviews を減らしてください。", count)
	if err != nil {
		return nil, err
	}
	result.SetError(errResponseTooLarge)
	return result, nil
}

// marshalToolResult は同じ JSON を構造化結果とテキストへ設定し、入力を変更しない。
func marshalToolResult(collection steam.Collection, code, message string, omitted int) (*mcp.CallToolResult, error) {
	data, err := jsonoutput.MarshalCollection(collection)
	if err != nil {
		return nil, fmt.Errorf("convert MCP collection: %w", err)
	}
	out := toolOutput{
		Collection: data, ReturnedCount: len(collection.Reviews), OmittedReviews: omitted,
		ErrorCode: code, Message: message,
		AnalysisGuide: analysisGuide{Version: analysisGuideVersion, Instructions: analysisGuideInstructions},
	}
	content, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("encode MCP result: %w", err)
	}
	return &mcp.CallToolResult{
		Content:           []mcp.Content{&mcp.TextContent{Text: string(content)}},
		StructuredContent: json.RawMessage(content),
		IsError:           code != "",
	}, nil
}
