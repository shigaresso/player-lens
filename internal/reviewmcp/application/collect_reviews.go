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
//
//contractcheck:data 収集・受付・応答変換の間で結果と原因を受け渡すデータ
type Result struct {
	Collection steamdomain.Collection
	Rejection  mcpdomain.RejectionReason
	Err        error
}

// CollectReviews は解析済みの条件で、利用枠と最大8秒の期限を管理して収集する。
// fetcher と gate は呼び出し側が用意した有効な依存を渡す。
// 入力の読み取りと MCP 用の条件検査は adapter が先に行う。
// 開始前キャンセルは収集を呼ばず、利用枠を消費せずに失敗結果を処理して戻る。
// 成功した確保は終了時に必ず返却する。
// 開始回数は戻さない。取得用 context はここで解放し、親をキャンセルしない。
// respond は非 nil の同期処理を渡す。成功・失敗・拒否の結果を一度だけ渡し、
// 応答生成の完了まで枠を保持する。収集の失敗は Result.Err で通知し、
// 戻り値には respond のエラーをそのまま返す。
func CollectReviews(ctx context.Context, fetcher steamapplication.PageFetcher, appID steamdomain.AppID, options steamdomain.CollectionOptions, gate *mcpdomain.Limiter, respond func(Result) error) error {
	if err := ctx.Err(); err != nil {
		collection := steamdomain.NewFailedCollection(steamdomain.Collection{
			AppID: appID, Options: options.Config(), Reviews: []steamdomain.Review{},
		})
		return respond(Result{Collection: collection, Err: fmt.Errorf("collect MCP reviews before start: %w", err)})
	}

	release, reason := gate.Acquire()
	if reason != mcpdomain.NotRejected {
		collection := steamdomain.NewFailedCollection(steamdomain.Collection{AppID: appID, Options: options.Config()})
		return respond(Result{Collection: collection, Rejection: reason})
	}

	defer release()

	ctx, cancel := context.WithTimeout(ctx, toolTimeout)
	defer cancel()

	collection, err := steamapplication.CollectReviews(ctx, fetcher, appID, options)
	return respond(Result{Collection: collection, Err: err})
}
