package adapter

import (
	"context"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	controller "github.com/shigaresso/player-lens/internal/reviewmcp/adapter/handler"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
)

const maxRequestBodyBytes = 16 * 1024

// NewHandler は読み取り専用のレビュー収集を Streamable HTTP で公開する。
// fetcher は生成成功した非 nil の取得依存で、キャンセル・期限を尊重するものを渡す。
// limits は domain.NewLimits または domain.DefaultLimits で生成し、生成失敗は呼び出し側で処理する。
// 利用回数はメモリ内に保持し、ハンドラーの作り直しでは引き継がない。
// HTTP クライアントと接続プールの終了は呼び出し側が担当する。
// config は NewHTTPConfig で生成し、ローカル用・公開用ともに渡す。
// 未生成の依存・設定は呼び出し側の契約違反とし、利用側では再検査しない。許可 Host から認証なしで受け付ける。
func NewHandler(fetcher steamapplication.PageFetcher, limits mcpdomain.Limits, config HTTPConfig) http.Handler {
	gate := mcpdomain.NewLimiter(limits, time.Now)
	return config.newHandler(func(request *http.Request) *mcp.Server {
		return newServer(fetcher, request.Context(), gate)
	})
}

func newServer(fetcher steamapplication.PageFetcher, requestContext context.Context, gate *mcpdomain.Limiter) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "PlayerLens", Version: "0.1.0"}, &mcp.ServerOptions{
		SupportedProtocolVersions: []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"},
	})
	server.AddTool(collectReviewsDefinition(), controller.NewCollectReviewsHandler(fetcher, requestContext, gate))
	return server
}
