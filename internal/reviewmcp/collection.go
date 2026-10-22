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
		return encodeResult(steam.Collection{}, "invalid_input", "取得条件が不正です。app_id とツールの入力上限を確認してください。", 0)
	}
	if ctx.Err() == nil {
		release, code := gate.acquire()
		if code != "" {
			failure := usageFailure(code)
			return encodeResult(steam.Collection{AppID: appID, Options: options, StopReason: steam.StopFailed}, failure.code, failure.message, 0)
		}
		defer release()
	}
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()
	collection, err := steam.CollectReviews(ctx, fetcher, appID, options)
	return encodeCollectionResult(collection, err)
}
