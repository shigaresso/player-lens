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

	var response *mcp.CallToolResult
	err = mcpapplication.CollectReviews(ctx, fetcher, appID, options, gate, func(result mcpapplication.Result) error {
		var encodeErr error
		if result.Rejection != mcpdomain.NotRejected {
			response, encodeErr = encodeResult(result.Collection, usageFailure(result.Rejection))
		} else {
			response, encodeErr = encodeCollectionResult(result.Collection, result.Err)
		}

		return encodeErr
	})
	return response, err
}
