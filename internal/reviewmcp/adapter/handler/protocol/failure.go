package protocol

import (
	"context"
	"errors"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
)

// toolFailureInfo は MCP ツールの失敗について公開するコードと説明文を表す。
// 元のエラーは呼び出し側が別途保持し、ゼロ値は公開する失敗情報がないことを表す。
type toolFailureInfo struct {
	code    string
	message string
}

func collectionFailure(err error) toolFailureInfo {
	switch {
	case err == nil:
		return toolFailureInfo{}
	case errors.Is(err, context.Canceled):
		return toolFailureInfo{"cancelled", "取得がキャンセルされました。取得済みの部分結果を返します。"}
	case errors.Is(err, context.DeadlineExceeded):
		return toolFailureInfo{"deadline_exceeded", "取得期限を超えました。件数やページ数を減らしてください。"}
	default:
		return toolFailureInfo{"fetch_failed", "レビューの取得に失敗しました。取得済みの部分結果を確認してください。"}
	}
}

// usageFailure は利用制限の拒否理由を公開情報へ変換する。
// 拒否なしではゼロ値を返し、未定義の内部値ではプログラミングエラーとして panic する。
func usageFailure(reason mcpdomain.RejectionReason) toolFailureInfo {
	switch reason {
	case mcpdomain.NotRejected:
		return toolFailureInfo{}
	case mcpdomain.DailyLimitReached:
		return toolFailureInfo{"daily_limit_reached", "収集開始回数の上限に達しました。利用枠が戻ってから再実行してください。"}
	case mcpdomain.MinuteLimitReached:
		return toolFailureInfo{"rate_limited", "収集開始回数の上限に達しました。利用枠が戻ってから再実行してください。"}
	case mcpdomain.ConcurrentLimitReached:
		return toolFailureInfo{"busy", "同時実行数の上限に達しました。処理の完了後に再実行してください。"}
	default:
		panic("unknown usage rejection reason")
	}
}

// CollectionErrorCode は公開応答と観測が共有する収集失敗のコードを返す。
// 成功は空文字とし、原因の表示文字列は返さない。
func CollectionErrorCode(err error) string {
	return collectionFailure(err).code
}

// UsageErrorCode は公開応答と観測が共有する受付拒否のコードを返す。
// 拒否なしは空文字とし、未知の内部値は usageFailure と同じく panic する。
func UsageErrorCode(reason mcpdomain.RejectionReason) string {
	return usageFailure(reason).code
}
