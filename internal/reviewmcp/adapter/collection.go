package adapter

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mcpapplication "github.com/shigaresso/player-lens/internal/reviewmcp/application"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func collect(ctx context.Context, fetcher steamapplication.PageFetcher, input json.RawMessage, gate *mcpdomain.Limiter) (*mcp.CallToolResult, error) {
	appID, options, err := parseInput(input)
	if err != nil {
		failure := toolFailureInfo{
			code:    "invalid_input",
			message: "取得条件が不正です。app_id とツールの入力上限を確認してください。",
		}
		return encodeResult(steamdomain.Collection{}, failure)
	}
	result := mcpapplication.CollectReviews(ctx, fetcher, appID, options, gate)
	if result.Rejection != mcpdomain.NotRejected {
		return encodeResult(result.Collection, usageFailure(result.Rejection))
	}
	return encodeCollectionResult(result.Collection, result.Err)
}
