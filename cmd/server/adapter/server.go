package adapter

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Serve は listener を所有し、終了時に接続と Serve の goroutine を回収する。
func Serve(ctx context.Context, listener net.Listener, handler http.Handler, timeout time.Duration) (err error) {
	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		// 終了シグナルではリクエストを即時中断せず、Shutdown の期限まで待つ。
		BaseContext: func(net.Listener) context.Context { return context.WithoutCancel(ctx) },
	}
	defer func() { err = errors.Join(err, server.Close()) }()

	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	select {
	case result := <-done:
		return serveError(result)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), timeout)
		defer cancel()

		shutdownErr := server.Shutdown(shutdownCtx)
		if shutdownErr != nil {
			shutdownErr = fmt.Errorf("HTTP サーバーの終了待ちに失敗した: %w", shutdownErr)
		}

		return errors.Join(shutdownErr, serveError(<-done))
	}
}

func serveError(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return fmt.Errorf("HTTP サーバーの実行に失敗した: %w", err)
}
