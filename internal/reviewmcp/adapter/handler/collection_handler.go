package handler

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
)

// newCollectReviewsHandler は HTTP の context を基に、SDK のキャンセルも収集へ伝える。
// 呼び出しごとに所有する context と連携 callback を終了時に解放する。
func newCollectReviewsHandler(fetcher steamapplication.PageFetcher, requestContext context.Context, gate *mcpdomain.Limiter) mcp.ToolHandler {
	return func(sdkContext context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, cancel := context.WithCancel(requestContext)
		stop := context.AfterFunc(sdkContext, cancel)
		defer stop()
		defer cancel()

		return collect(ctx, fetcher, request.Params.Arguments, gate)
	}
}
