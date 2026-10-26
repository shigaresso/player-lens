package handler

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/handler/protocol"
	mcpapplication "github.com/shigaresso/player-lens/internal/reviewmcp/application"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
)

// NewCollectReviewsHandler は HTTP の context を基に、SDK のキャンセルも収集へ伝える。
// 呼び出しごとに所有する context と連携 callback を終了時に解放する。
// fetcher・requestContext・gate は呼び出し側が有効なものを渡す。
// gate は SDK handler 全体で共有し、呼び出しごとに作り直さない。
func NewCollectReviewsHandler(fetcher steamapplication.PageFetcher, requestContext context.Context, gate *mcpdomain.Limiter) mcp.ToolHandler {
	return func(sdkContext context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, cancel := context.WithCancel(requestContext)
		stop := context.AfterFunc(sdkContext, cancel)
		defer stop()
		defer cancel()

		return collect(ctx, fetcher, request.Params.Arguments, gate)
	}
}

func collect(ctx context.Context, fetcher steamapplication.PageFetcher, input json.RawMessage, gate *mcpdomain.Limiter) (*mcp.CallToolResult, error) {
	appID, options, err := protocol.ParseInput(input)
	if err != nil {
		return protocol.InvalidInputResult()
	}

	var response *mcp.CallToolResult
	err = mcpapplication.CollectReviews(ctx, fetcher, appID, options, gate, func(result mcpapplication.Result) error {
		var encodeErr error
		response, encodeErr = protocol.EncodeResult(result)

		return encodeErr
	})
	return response, err
}
