package reviewmcp

import (
	"context"
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
}

func collect(ctx context.Context, fetcher steam.PageFetcher, input json.RawMessage) (*mcp.CallToolResult, error) {
	appID, options, err := parseInput(input)
	if err != nil {
		return encodeResult(steam.Collection{}, "invalid_input", "取得条件が不正です。app_id とツールの入力上限を確認してください。", 0)
	}
	collection, err := steam.CollectReviews(ctx, fetcher, appID, options)
	if err != nil {
		code := "fetch_failed"
		message := "レビューの取得に失敗しました。取得済みの部分結果を確認してください。"
		if errors.Is(err, context.Canceled) {
			code = "cancelled"
			message = "取得がキャンセルされました。取得済みの部分結果を返します。"
		} else if errors.Is(err, context.DeadlineExceeded) {
			code = "deadline_exceeded"
			message = "取得期限を超えました。件数やページ数を減らしてください。"
		}
		result, encodeErr := encodeResult(collection, code, message, 0)
		if encodeErr != nil {
			return nil, encodeErr
		}
		result.SetError(err)
		return result, nil
	}
	return encodeResult(collection, "", "", 0)
}

func encodeResult(collection steam.Collection, code, message string, omitted int) (*mcp.CallToolResult, error) {
	data, err := jsonoutput.MarshalCollection(collection)
	if err != nil {
		return nil, fmt.Errorf("convert MCP collection: %w", err)
	}
	out := toolOutput{Collection: data, ReturnedCount: len(collection.Reviews), OmittedReviews: omitted, ErrorCode: code, Message: message}
	content, err := json.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("encode MCP result: %w", err)
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(content)}}, StructuredContent: json.RawMessage(content), IsError: code != ""}, nil
}
