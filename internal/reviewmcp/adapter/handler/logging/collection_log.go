// Package logging は MCP レビュー収集の完了ログと安全な原因分類を担当する。
package logging

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/handler/protocol"
	mcpapplication "github.com/shigaresso/player-lens/internal/reviewmcp/application"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// ReviewCollection は Controller の開始から応答生成までの1回の実行結果を表す。
// AppID は解析成功時だけ設定し、Result は UseCase が返った場合だけ渡す。
// Stage は panic の発生箇所を表し、input・collection・response_encode の固定値を使う。
// エラーは分類のみに使い、入力・本文・カーソル・エラー文字列をログへ含めない。
type ReviewCollection struct {
	Duration         time.Duration
	AppID            steamdomain.AppID
	InputErr         error
	Result           *mcpapplication.Result
	ResponseErr      error
	ResponseTooLarge bool
	Panicked         bool
	Stage            string
}

type completionStatus struct {
	level   slog.Level
	outcome string
	stage   string
	code    string
	kind    string
}

// Write は完了イベントを1回記録し、出力先の Write エラーでは呼び出し結果を変更しない。
// ctx と logger は呼び出し側から渡し、送信成功・保存成功は保証しない。
func (entry ReviewCollection) Write(ctx context.Context, logger *slog.Logger) {
	status := entry.status()
	attrs := []slog.Attr{
		slog.String("event", "review_collection.completed"),
		slog.Float64("duration_ms", float64(entry.Duration)/float64(time.Millisecond)),
		slog.String("outcome", status.outcome),
	}
	if status.stage != "" {
		attrs = append(attrs, slog.String("failure_stage", status.stage), slog.String("error_code", status.code), slog.String("cause_kind", status.kind))
	}

	if entry.AppID != 0 {
		attrs = append(attrs, slog.Uint64("app_id", uint64(entry.AppID)))
	}

	if entry.Result != nil && entry.Result.Rejection == mcpdomain.NotRejected {
		collection := entry.Result.Collection
		attrs = append(attrs, slog.Int("collected_count", len(collection.Reviews)), slog.Int("pages_fetched", collection.PagesFetched), slog.String("stop_reason", string(collection.StopReason)))
	}

	logger.LogAttrs(ctx, status.level, "MCP のレビュー収集を処理した", attrs...)
}

func (entry ReviewCollection) status() completionStatus {
	switch {
	case entry.Panicked:
		return completionStatus{level: slog.LevelError, outcome: "failed", stage: entry.Stage, code: "panic", kind: "panic"}
	case entry.ResponseErr != nil:
		code := "response_encoding_failed"
		if errors.Is(entry.ResponseErr, protocol.ErrResponseTooLarge) {
			code = "response_too_large"
		}

		return completionStatus{level: slog.LevelError, outcome: "failed", stage: "response_encode", code: code, kind: causeKind(entry.ResponseErr)}
	case entry.ResponseTooLarge:
		return completionStatus{level: slog.LevelWarn, outcome: "failed", stage: "response_encode", code: "response_too_large", kind: "response_too_large"}
	case entry.InputErr != nil:
		return completionStatus{level: slog.LevelInfo, outcome: "rejected", stage: "input", code: "invalid_input", kind: causeKind(entry.InputErr)}
	case entry.Result == nil:
		return completionStatus{level: slog.LevelError, outcome: "failed", stage: entry.Stage, code: "missing_result", kind: "unknown"}
	case entry.Result.Rejection != mcpdomain.NotRejected:
		return completionStatus{level: slog.LevelInfo, outcome: "rejected", stage: "collection", code: protocol.UsageErrorCode(entry.Result.Rejection), kind: "usage_limit"}
	case entry.Result.Err != nil:
		status := completionStatus{level: slog.LevelWarn, outcome: "failed", stage: "collection", code: protocol.CollectionErrorCode(entry.Result.Err), kind: causeKind(entry.Result.Err)}
		if status.code == "cancelled" {
			status.level, status.outcome = slog.LevelInfo, "cancelled"
		}

		return status
	default:
		return completionStatus{level: slog.LevelInfo, outcome: "success"}
	}
}

func causeKind(err error) string {
	var syntaxError *json.SyntaxError
	var typeError *json.UnmarshalTypeError
	var statusError *steamadapter.HTTPStatusError
	var networkError net.Error
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.As(err, &syntaxError):
		return "json_syntax"
	case errors.As(err, &typeError):
		return "json_type"
	case errors.Is(err, protocol.ErrResponseTooLarge):
		return "response_too_large"
	case errors.Is(err, steamdomain.ErrCursorCycle):
		return "cursor_cycle"
	case errors.As(err, &statusError):
		return "http_status"
	case errors.As(err, &networkError) && networkError.Timeout():
		return "timeout"
	case errors.Is(err, steamadapter.ErrResponseTooLarge):
		return "upstream_response_too_large"
	case errors.Is(err, steamadapter.ErrInvalidResponse):
		return "invalid_response"
	case errors.Is(err, steamdomain.ErrInvalidArgument):
		return "invalid_argument"
	default:
		return "unknown"
	}
}
