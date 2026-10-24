package reviewmcp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shigaresso/player-lens/internal/mcpaccess"
	"github.com/shigaresso/player-lens/internal/steam"
)

const maxRequestBodyBytes = 16 * 1024

// NewHandler は読み取り専用のレビュー収集を Streamable HTTP で公開する。
// fetcher はキャンセル・期限を尊重する取得依存とする。nil はエラーとする。
// limits は NewLimits または DefaultLimits で生成し、ゼロ値はエラーとする。
// 利用回数はメモリ内に保持し、ハンドラーの作り直しでは引き継がない。
// HTTP クライアントと接続プールの終了は呼び出し側が担当する。
// access が nil ならローカル専用、指定時は許可 Host から認証なしで受け付ける。
func NewHandler(fetcher steam.PageFetcher, limits Limits, access *mcpaccess.Access) (http.Handler, error) {
	if fetcher == nil {
		return nil, fmt.Errorf("prepare review MCP handler: %w", steam.ErrInvalidArgument)
	}
	if limits == (Limits{}) {
		return nil, fmt.Errorf("prepare review MCP handler: limits are required: %w", steam.ErrInvalidArgument)
	}
	gate := newLimiter(limits, time.Now)
	handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return newServer(fetcher, request.Context(), gate)
	}, &mcp.StreamableHTTPOptions{
		Stateless:                    true,
		JSONResponse:                 true,
		PropagateRequestCancellation: true,
		MaxRequestBodyBytes:          maxRequestBodyBytes,
		DisableLocalhostProtection:   access != nil,
	})
	if access != nil {
		return access.Protect(handler)
	}
	return mcpaccess.ProtectLocal(handler), nil
}

func newServer(fetcher steam.PageFetcher, requestContext context.Context, gate *limiter) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{Name: "PlayerLens", Version: "0.1.0"}, &mcp.ServerOptions{
		SupportedProtocolVersions: []string{"2026-07-28", "2025-11-25", "2025-06-18", "2025-03-26"},
	})
	server.AddTool(collectReviewsDefinition(), newCollectReviewsHandler(fetcher, requestContext, gate))
	return server
}
