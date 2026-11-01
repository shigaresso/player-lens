// Package logging はプロンプト作成画面の応答完了ログを担当する。
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

// PromptPage はサーバー側の1回の画面応答の処理結果を表す。
// Route は固定パス、Duration は応答処理の時間で、クライアントの表示成功を表さない。
type PromptPage struct {
	Route      string
	Method     string
	StatusCode int
	Duration   time.Duration
	WriteErr   error
}

// Write は完了イベントを1回記録する。出力先のエラーで応答を変更しない。
// ctx と logger は呼び出し側から渡し、入力値や仮の trace ID は記録しない。
func (entry PromptPage) Write(ctx context.Context, logger *slog.Logger) {
	level, outcome := slog.LevelInfo, "success"
	if entry.StatusCode >= http.StatusInternalServerError {
		level, outcome = slog.LevelError, "response_failed"
	} else if entry.StatusCode >= http.StatusBadRequest {
		level, outcome = slog.LevelWarn, "rejected"
	}

	if entry.WriteErr != nil {
		level, outcome = slog.LevelError, "write_failed"
	}

	attrs := []slog.Attr{
		slog.String("event", "prompt_page.completed"),
		slog.String("http.route", entry.Route),
		slog.String("http.request.method", entry.Method),
		slog.Int("http.response.status_code", entry.StatusCode),
		slog.Float64("duration_ms", float64(entry.Duration)/float64(time.Millisecond)),
		slog.String("outcome", outcome),
	}
	if entry.WriteErr != nil {
		attrs = append(attrs, slog.String("error_code", "response_write_failed"), slog.String("cause_kind", writeCause(entry.WriteErr)))
	}

	logger.LogAttrs(ctx, level, "プロンプト作成画面の応答を処理した", attrs...)
}

func writeCause(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, io.ErrClosedPipe), errors.Is(err, net.ErrClosed), errors.Is(err, syscall.EPIPE), errors.Is(err, syscall.ECONNRESET):
		return "connection_closed"
	default:
		var networkError net.Error
		if errors.As(err, &networkError) && networkError.Timeout() {
			return "timeout"
		}

		return "unknown"
	}
}
