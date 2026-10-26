package protocol

import (
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
)

import "testing"

func TestUsageFailure(t *testing.T) {
	const budgetMessage = "収集開始回数の上限に達しました。利用枠が戻ってから再実行してください。"
	for _, tc := range []struct {
		name    string
		reason  mcpdomain.RejectionReason
		want    toolFailureInfo
		invalid bool
	}{
		{"拒否なしでは公開する失敗情報を返さない", mcpdomain.NotRejected, toolFailureInfo{}, false},
		{"日次の上限を既存の公開情報に変換する", mcpdomain.DailyLimitReached, toolFailureInfo{"daily_limit_reached", budgetMessage}, false},
		{"分次の上限を既存の公開情報に変換する", mcpdomain.MinuteLimitReached, toolFailureInfo{"rate_limited", budgetMessage}, false},
		{"同時数の上限を完了待ちの公開情報に変換する", mcpdomain.ConcurrentLimitReached, toolFailureInfo{"busy", "同時実行数の上限に達しました。処理の完了後に再実行してください。"}, false},
		{"負の内部値はプログラミングエラーとする", mcpdomain.RejectionReason(-1), toolFailureInfo{}, true},
		{"未定義の内部値はプログラミングエラーとする", mcpdomain.RejectionReason(4), toolFailureInfo{}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.invalid {
				defer func() {
					if recover() == nil {
						t.Fatal("未定義の拒否理由を受け付けた")
					}
				}()

				usageFailure(tc.reason)
				return
			}

			if got := usageFailure(tc.reason); got != tc.want {
				t.Fatalf("info=%+v want=%+v", got, tc.want)
			}
		})
	}
}
