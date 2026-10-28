package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil)).With("service.name", "player-lens")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx, os.Getenv("PORT"), logger)
	stop()
	if err != nil {
		logger.ErrorContext(ctx, "HTTP サーバーの実行に失敗した",
			slog.String("event", "server.failed"), slog.Any("error", err))
		os.Exit(1)
	}
}
