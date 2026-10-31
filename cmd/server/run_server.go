package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	serveradapter "github.com/shigaresso/player-lens/cmd/server/adapter"
	serverhandler "github.com/shigaresso/player-lens/cmd/server/adapter/handler"
	reviewmcp "github.com/shigaresso/player-lens/internal/reviewmcp/adapter"
)

const shutdownTimeout = 10 * time.Second

func run(ctx context.Context, port string, logger *slog.Logger) error {
	portNumber, err := serveradapter.NewValidatedPortNumber(port)
	if err != nil {
		return err
	}

	limits, err := serveradapter.ReadMCPLimits(os.Getenv)
	if err != nil {
		return fmt.Errorf("MCP 利用制限の設定に失敗した: %w", err)
	}

	mcpConfig, err := serveradapter.ReadMCPHTTPConfig(os.Getenv)
	if err != nil {
		return fmt.Errorf("MCP 公開設定に失敗した: %w", err)
	}

	fetcher, closeIdleConnections, err := serveradapter.NewSteamFetcher()
	if err != nil {
		return fmt.Errorf("レビュー取得クライアントの準備に失敗した: %w", err)
	}
	defer closeIdleConnections()

	mcpHandler, err := reviewmcp.NewHandler(fetcher, limits, mcpConfig, logger, time.Now)
	if err != nil {
		return fmt.Errorf("MCP サーバーの準備に失敗した: %w", err)
	}

	healthHandler := serverhandler.NewHealthHandler(logger, time.Now)
	router := http.NewServeMux()
	serveradapter.RegisterRoutes(router, healthHandler, mcpHandler)

	address := net.JoinHostPort("", portNumber.String())
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("HTTP の待ち受けに失敗した: %w", err)
	}

	logger.InfoContext(ctx, "HTTP サーバーを起動した",
		slog.String("event", "server.started"), slog.String("address", address))
	return serveradapter.Serve(ctx, listener, router, shutdownTimeout)
}
