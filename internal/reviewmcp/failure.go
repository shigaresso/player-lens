package reviewmcp

import (
	"context"
	"errors"
)

// toolFailure はモデルへ公開するコードと説明文を表す。内部の原因は含めない。
type toolFailure struct {
	code    string
	message string
}

func collectionFailure(err error) toolFailure {
	switch {
	case err == nil:
		return toolFailure{}
	case errors.Is(err, context.Canceled):
		return toolFailure{"cancelled", "取得がキャンセルされました。取得済みの部分結果を返します。"}
	case errors.Is(err, context.DeadlineExceeded):
		return toolFailure{"deadline_exceeded", "取得期限を超えました。件数やページ数を減らしてください。"}
	default:
		return toolFailure{"fetch_failed", "レビューの取得に失敗しました。取得済みの部分結果を確認してください。"}
	}
}

func usageFailure(code string) toolFailure {
	if code == "busy" {
		return toolFailure{code, "同時実行数の上限に達しました。処理の完了後に再実行してください。"}
	}
	return toolFailure{code, "収集開始回数の上限に達しました。利用枠が戻ってから再実行してください。"}
}
