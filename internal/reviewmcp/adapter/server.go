package adapter

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	accessadapter "github.com/shigaresso/player-lens/internal/mcpaccess/adapter"
	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
	controller "github.com/shigaresso/player-lens/internal/reviewmcp/adapter/handler"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

const maxRequestBodyBytes = 16 * 1024

// NewHandler は読み取り専用のレビュー収集を Streamable HTTP で公開する。
// fetcher はキャンセル・期限を尊重する取得依存とする。nil はエラーとする。
// limits は domain.NewLimits または domain.DefaultLimits で生成し、ゼロ値はエラーとする。
// 利用回数はメモリ内に保持し、ハンドラーの作り直しでは引き継がない。
// HTTP クライアントと接続プールの終了は呼び出し側が担当する。
// config は NewHTTPConfig で生成し、ローカル用・公開用ともに渡す。
// config のゼロ値は構築エラーとし、許可 Host から認証なしで受け付ける。
// logger と計測用 now は非 nil かつ並行利用可能なものを渡す。nil は構築エラーとする。
func NewHandler(fetcher steamapplication.PageFetcher, limits mcpdomain.Limits, config HTTPConfig, logger *slog.Logger, now func() time.Time) (http.Handler, error) {
	if fetcher == nil {
		return nil, fmt.Errorf("prepare review MCP handler: %w", steamdomain.ErrInvalidArgument)
	}

	if limits == (mcpdomain.Limits{}) {
		return nil, fmt.Errorf("prepare review MCP handler: limits are required: %w", steamdomain.ErrInvalidArgument)
	}

	if config.access == nil {
		return nil, fmt.Errorf("prepare review MCP handler: HTTP config is required: %w", mcpaccess.ErrInvalidConfig)
	}

	if logger == nil || now == nil {
		return nil, fmt.Errorf("prepare review MCP handler: logger and clock are required: %w", steamdomain.ErrInvalidArgument)
	}

	sharedLimiter := mcpdomain.NewLimiter(limits, time.Now)

	createServerForRequest := func(request *http.Request) *mcp.Server {
		return newServer(fetcher, request.Context(), sharedLimiter, logger, now)
	}

	sdkHTTPHandler := mcp.NewStreamableHTTPHandler(createServerForRequest, &config.sdkOptions)

	return accessadapter.Protect(config.access, config.deniedMessage, sdkHTTPHandler)
}

func newServer(fetcher steamapplication.PageFetcher, requestContext context.Context, gate *mcpdomain.Limiter, logger *slog.Logger, now func() time.Time) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "PlayerLens", Version: "0.1.0"}, &mcp.ServerOptions{
		SupportedProtocolVersions: []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"},
	})
	server.AddTool(collectReviewsDefinition(), controller.NewCollectReviewsHandler(fetcher, requestContext, gate, logger, now))
	return server
}
