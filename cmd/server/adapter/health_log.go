package adapter

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"syscall"
	"time"
)

type healthCheckLog struct {
	method   string
	duration time.Duration
	writeErr error
}

func (entry healthCheckLog) write(ctx context.Context, logger *slog.Logger) {
	attrs := []slog.Attr{
		slog.String("event", "health_check.completed"),
		slog.String("http.request.method", entry.method),
		slog.String("http.route", healthPath),
		slog.Int("http.response.status_code", http.StatusOK),
		slog.Float64("duration_ms", float64(entry.duration)/float64(time.Millisecond)),
	}
	level, outcome := slog.LevelInfo, "success"
	if entry.writeErr != nil {
		level, outcome = slog.LevelError, "write_failed"
		attrs = append(attrs,
			slog.String("error_code", "response_write_failed"),
			slog.String("cause_kind", healthWriteCause(entry.writeErr)))
	}

	attrs = append(attrs, slog.String("outcome", outcome))
	logger.LogAttrs(ctx, level, "ヘルスチェックを処理した", attrs...)
}

func healthWriteCause(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, io.ErrClosedPipe), errors.Is(err, net.ErrClosed),
		errors.Is(err, syscall.EPIPE), errors.Is(err, syscall.ECONNRESET):
		return "connection_closed"
	default:
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			return "timeout"
		}

		return "unknown"
	}
}
