package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	serveradapter "github.com/shigaresso/player-lens/cmd/server/adapter"
	reviewmcp "github.com/shigaresso/player-lens/internal/reviewmcp/adapter"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Getenv("PORT"))
	stop()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, port string) error {
	address, err := serveradapter.ListenAddress(port)
	if err != nil {
		return err
	}
	limits, err := serveradapter.ReadMCPLimits(os.Getenv)
	if err != nil {
		return fmt.Errorf("MCP 利用制限の設定に失敗した: %w", err)
	}
	access, err := serveradapter.ReadMCPAccess(os.Getenv)
	if err != nil {
		return fmt.Errorf("MCP 公開設定に失敗した: %w", err)
	}
	fetcher, closeIdleConnections, err := serveradapter.NewSteamFetcher()
	if err != nil {
		return fmt.Errorf("レビュー取得クライアントの準備に失敗した: %w", err)
	}
	defer closeIdleConnections()
	mcpHandler, err := reviewmcp.NewHandler(fetcher, limits, access)
	if err != nil {
		return fmt.Errorf("MCP サーバーの準備に失敗した: %w", err)
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("HTTP の待ち受けに失敗した: %w", err)
	}
	log.Printf("HTTP サーバーを起動した: %s", address)
	return serveradapter.Serve(ctx, listener, serveradapter.NewHandler(mcpHandler), 10*time.Second)
}
