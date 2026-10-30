package application

import (
	"context"
	"fmt"
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
// 開始前キャンセルは収集を呼ばず、利用枠を消費せずに失敗結果を返す。
// 確保した同時枠と取得用 context は、結果を返す前に解放する。
// 開始回数は戻さない。取得用 context はここで解放し、親をキャンセルしない。
// 収集の失敗は Result.Err、受付の拒否は Result.Rejection に保持する。
// 呼び出し側が結果を応答へ変換し、その処理は同時枠の保持範囲に含めない。
func CollectReviews(ctx context.Context, fetcher steamapplication.PageFetcher, appID steamdomain.AppID, options steamdomain.CollectionOptions, gate *mcpdomain.Limiter) Result {
	if err := ctx.Err(); err != nil {
		collection := steamdomain.NewFailedCollection(steamdomain.Collection{
			AppID: appID, Options: options.Config(), Reviews: []steamdomain.Review{},
		})
		return Result{Collection: collection, Err: fmt.Errorf("collect MCP reviews before start: %w", err)}
	}

	release, reason := gate.Acquire()
	if reason != mcpdomain.NotRejected {
		collection := steamdomain.NewFailedCollection(steamdomain.Collection{AppID: appID, Options: options.Config()})
		return Result{Collection: collection, Rejection: reason}
	}

	defer release()

	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()

	collection, err := steamapplication.CollectReviews(ctx, fetcher, appID, options)
	return Result{Collection: collection, Err: err}
}
