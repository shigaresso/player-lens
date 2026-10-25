package application

import (
	"context"
	"time"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

const toolTimeout = 8 * time.Second

// Result は収集結果、受付の拒否理由、取得の失敗を分けて保持する。
// 拒否時は Err が nil で、Collection に入力条件と失敗の終了理由を保持する。
// 取得の失敗は Err に元の原因と Collection に部分結果を保持する。
type Result struct {
	Collection steamdomain.Collection
	Rejection  mcpdomain.RejectionReason
	Err        error
}

// CollectReviews は解析済みの条件で、利用枠と最大8秒の期限を管理して収集する。
// fetcher と gate は呼び出し側が用意した有効な依存を渡す。
// 入力の読み取りと MCP 用の条件検査は adapter が先に行う。
// 開始前キャンセルでは利用枠を消費せず、成功した確保は終了時に必ず返却する。
// 開始回数は戻さない。取得用 context はここで解放し、親をキャンセルしない。
// respond は非 nil の同期処理を渡す。成功・失敗・拒否の結果を一度だけ渡し、
// 応答生成の完了まで枠を保持する。収集の失敗は Result.Err で通知し、
// 戻り値には respond のエラーをそのまま返す。
func CollectReviews(ctx context.Context, fetcher steamapplication.PageFetcher, appID steamdomain.AppID, options steamdomain.CollectionOptions, gate *mcpdomain.Limiter, respond func(Result) error) error {
	if ctx.Err() == nil {
		release, reason := gate.Acquire()
		if reason != mcpdomain.NotRejected {
			return respond(Result{Collection: steamdomain.Collection{AppID: appID, Options: options.Config(), StopReason: steamdomain.StopFailed}, Rejection: reason})
		}
		defer release()
	}
	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()
	collection, err := steamapplication.CollectReviews(ctx, fetcher, appID, options)
	return respond(Result{Collection: collection, Err: err})
}
