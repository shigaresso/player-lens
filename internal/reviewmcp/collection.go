package reviewmcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/steam"
)

const toolTimeout = 8 * time.Second

func collect(ctx context.Context, fetcher steam.PageFetcher, input json.RawMessage, gate *limiter) (*mcp.CallToolResult, error) {
	appID, options, err := parseInput(input)
	if err != nil {
		failure := toolFailureInfo{
			code:    "invalid_input",
			message: "取得条件が不正です。app_id とツールの入力上限を確認してください。",
		}
		return encodeResult(steam.Collection{}, failure)
	}
	if ctx.Err() == nil {
		release, reason := gate.acquire()
		if reason != usageNotRejected {
			failure := usageFailure(reason)
			return encodeResult(steam.Collection{AppID: appID, Options: options, StopReason: steam.StopFailed}, failure)
		}
		defer release()
	}
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()
	collection, err := steam.CollectReviews(ctx, fetcher, appID, options)
	return encodeCollectionResult(collection, err)
}
