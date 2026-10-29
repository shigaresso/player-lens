package logging

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

// HealthCheck は1回の生存確認の応答処理結果を表す。
// Route は登録時の固定パスを渡し、Duration は応答処理の所要時間とする。
//
//contractcheck:data HTTP 完了時の状態と原因をログへ渡すデータ
type HealthCheck struct {
	Route    string
	Method   string
	Duration time.Duration
	WriteErr error
}

// Write は結果を構造化ログへ記録し、出力先のエラーでは応答を変更しない。
func (entry HealthCheck) Write(ctx context.Context, logger *slog.Logger) {
	attrs := []slog.Attr{
		slog.String("event", "health_check.completed"),
		slog.String("http.request.method", entry.Method),
		slog.String("http.route", entry.Route),
		slog.Int("http.response.status_code", http.StatusOK),
		slog.Float64("duration_ms", float64(entry.Duration)/float64(time.Millisecond)),
	}
	level, outcome := slog.LevelInfo, "success"
	if entry.WriteErr != nil {
		level, outcome = slog.LevelError, "write_failed"
		attrs = append(attrs,
			slog.String("error_code", "response_write_failed"),
			slog.String("cause_kind", healthWriteCause(entry.WriteErr)))
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
