package reviewmcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/steam"
)

// newCollectReviewsHandler は HTTP の context を基に、SDK のキャンセルも収集へ伝える。
// 呼び出しごとに所有する context と連携 callback を終了時に解放する。
func newCollectReviewsHandler(fetcher steam.PageFetcher, requestContext context.Context, gate *limiter) mcp.ToolHandler {
	return func(sdkContext context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, cancel := context.WithCancel(requestContext)
		stop := context.AfterFunc(sdkContext, cancel)
		defer stop()
		defer cancel()
		return collect(ctx, fetcher, request.Params.Arguments, gate)
	}
}
