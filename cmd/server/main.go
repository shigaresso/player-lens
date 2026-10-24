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

	"github.com/shigaresso/player-lens/internal/reviewmcp"
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
	address, err := listenAddress(port)
	if err != nil {
		return err
	}
	limits, err := readMCPLimits(os.Getenv)
	if err != nil {
		return fmt.Errorf("MCP 利用制限の設定に失敗した: %w", err)
	}
	access, err := readMCPAccess(os.Getenv)
	if err != nil {
		return fmt.Errorf("MCP 公開設定に失敗した: %w", err)
	}
	fetcher, closeIdleConnections, err := newSteamFetcher()
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
	return serve(ctx, listener, newHandler(mcpHandler), 10*time.Second)
}
