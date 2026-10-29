package handler

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/handler/protocol"
	mcpapplication "github.com/shigaresso/player-lens/internal/reviewmcp/application"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
)

// collectReviewsHandler は HTTP リクエストごとの固定の依存を保持する。
// 呼び出しごとの context は Handle 内で所有し、フィールドを書き換えない。
type collectReviewsHandler struct {
	fetcher        steamapplication.PageFetcher
	requestContext context.Context
	gate           *mcpdomain.Limiter
}

// NewCollectReviewsHandler はレビュー収集の Controller を組み立て、SDK 用のメソッド値を返す。
// HTTP リクエストごとに構築し、fetcher・requestContext・gate は有効なものを渡す。
// gate は SDK handler 全体で共有し、呼び出しごとに作り直さない。
func NewCollectReviewsHandler(fetcher steamapplication.PageFetcher, requestContext context.Context, gate *mcpdomain.Limiter) mcp.ToolHandler {
	handler := collectReviewsHandler{fetcher: fetcher, requestContext: requestContext, gate: gate}
	return handler.Handle
}

// Handle は HTTP と SDK のキャンセルを連携し、入力・収集・応答変換を順に行う。
// 正常・失敗・panic による終了時も所有 context と連携 callback を解放し、親はキャンセルしない。
func (h collectReviewsHandler) Handle(sdkContext context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ctx, cancel := context.WithCancel(h.requestContext)
	stop := context.AfterFunc(sdkContext, cancel)
	defer stop()
	defer cancel()

	appID, options, err := protocol.ParseInput(request.Params.Arguments)
	if err != nil {
		return protocol.InvalidInputResult()
	}

	var response *mcp.CallToolResult
	err = mcpapplication.CollectReviews(ctx, h.fetcher, appID, options, h.gate, func(result mcpapplication.Result) error {
		var encodeErr error
		response, encodeErr = protocol.EncodeResult(result)

		return encodeErr
	})
	return response, err
}
