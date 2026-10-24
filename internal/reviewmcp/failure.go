package reviewmcp

import (
	"context"
	"errors"
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

func usageFailure(code string) toolFailureInfo {
	if code == "busy" {
		return toolFailureInfo{code, "同時実行数の上限に達しました。処理の完了後に再実行してください。"}
	}
	return toolFailureInfo{code, "収集開始回数の上限に達しました。利用枠が戻ってから再実行してください。"}
}
