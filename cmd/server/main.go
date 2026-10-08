package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shigaresso/player-lens/internal/reviewmcp"
	"github.com/shigaresso/player-lens/internal/steam"
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
	transport := http.DefaultTransport.(*http.Transport).Clone()
	defer transport.CloseIdleConnections()
	fetcher, err := steam.NewClient(&http.Client{Transport: transport, Timeout: 10 * time.Second}, "")
	if err != nil {
		return fmt.Errorf("Steam クライアントの準備に失敗した: %w", err)
	}
	mcpHandler, err := reviewmcp.NewHandler(fetcher)
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
