package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/handler/logging"
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
	logger         *slog.Logger
	now            func() time.Time
}

// NewCollectReviewsHandler はレビュー収集の Controller を組み立て、SDK 用のメソッド値を返す。
// HTTP リクエストごとに構築し、fetcher・requestContext・gate は有効なものを渡す。
// gate は SDK handler 全体で共有し、呼び出しごとに作り直さない。
// logger と計測用の now は構築側で検査済みの非 nil かつ並行利用可能な依存を渡す。
func NewCollectReviewsHandler(fetcher steamapplication.PageFetcher, requestContext context.Context, gate *mcpdomain.Limiter, logger *slog.Logger, now func() time.Time) mcp.ToolHandler {
	handler := collectReviewsHandler{fetcher: fetcher, requestContext: requestContext, gate: gate, logger: logger, now: now}
	return handler.Handle
}

// Handle は HTTP と SDK のキャンセルを連携し、入力・収集・応答変換を順に行う。
// 正常・失敗・panic による終了時も所有 context と連携 callback を解放し、親はキャンセルしない。
// 応答生成までの結果を1回記録し、送信完了とは区別する。panic は回復せず値もログへ含めない。
func (h collectReviewsHandler) Handle(sdkContext context.Context, request *mcp.CallToolRequest) (response *mcp.CallToolResult, err error) {
	started := h.now()
	ctx, cancel := context.WithCancel(h.requestContext)
	stop := context.AfterFunc(sdkContext, cancel)
	defer stop()
	defer cancel()

	entry := logging.ReviewCollection{Stage: "input"}
	completed := false
	defer func() {
		entry.Panicked = !completed
		if completed {
			entry.ResponseErr = err
			if response != nil {
				entry.ResponseTooLarge = errors.Is(response.GetError(), protocol.ErrResponseTooLarge)
			}
		}

		entry.Duration = h.now().Sub(started)
		entry.Write(ctx, h.logger)
	}()

	appID, options, err := protocol.ParseInput(request.Params.Arguments)
	if err != nil {
		entry.InputErr = err
		entry.Stage = "response_encode"
		response, err = protocol.InvalidInputResult(err)
		completed = true
		return response, err
	}

	entry.AppID = appID
	entry.Stage = "collection"
	result := mcpapplication.CollectReviews(ctx, h.fetcher, appID, options, h.gate)

	entry.Result = &result
	entry.Stage = "response_encode"
	response, err = protocol.EncodeResult(result)
	completed = true
	return response, err
}
