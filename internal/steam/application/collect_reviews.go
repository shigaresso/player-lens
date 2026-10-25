package application

import (
	"context"
	"fmt"
	"time"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// PageFetcher は取得条件から1ページと次のカーソルを返す依存を表す。
// 空ページは成功、不正入力や取得失敗は原因を保持したエラーとする。
// 渡された context のキャンセルと期限を尊重する。
// 通信と応答 Body の解放は実装側、接続プールの終了は組み立てた呼び出し側が担当する。
type PageFetcher interface {
	FetchReviewsPage(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error)
}

// CollectReviews は先頭から上限付きでページを取得する。再試行と保存は行わない。
// fetcher は有効な取得依存を渡す。入力不正ではゼロ値と ErrInvalidArgument を返す。
// 呼び出し側の短い期限を優先し、収集全体を最大2分とする。
// 取得・循環・context の失敗では部分結果と StopFailed、原因を保持したエラーを返す。
func CollectReviews(ctx context.Context, fetcher PageFetcher, appID steamdomain.AppID, options steamdomain.CollectionOptions) (steamdomain.Collection, error) {
	if fetcher == nil || appID == 0 {
		return steamdomain.Collection{}, fmt.Errorf("page fetcher and positive AppID are required: %w", steamdomain.ErrInvalidArgument)
	}
	options, err := steamdomain.NormalizeCollectionOptions(options)
	if err != nil {
		return steamdomain.Collection{}, fmt.Errorf("prepare Steam review collection: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	result := steamdomain.Collection{AppID: appID, Options: options, Reviews: []steamdomain.Review{}}
	seenIDs := make(map[steamdomain.RecommendationID]bool)
	seenCursors := map[string]bool{steamdomain.FirstPageCursor: true}
	cursor := steamdomain.FirstPageCursor
	for result.PagesFetched < options.MaxPages {
		if err := ctx.Err(); err != nil {
			result.StopReason = steamdomain.StopFailed
			return result, fmt.Errorf("collect Steam reviews before page %d: %w", result.PagesFetched+1, err)
		}
		size := min(options.PageSize, options.MaxReviews-len(result.Reviews))
		page, err := fetcher.FetchReviewsPage(ctx, appID, steamdomain.PageOptions{ReviewType: options.ReviewType, Language: options.Language, Cursor: cursor, PageSize: size})
		if err != nil {
			result.StopReason = steamdomain.StopFailed
			return result, fmt.Errorf("collect Steam review page %d: %w", result.PagesFetched+1, err)
		}
		result.PagesFetched++
		if len(page.Reviews) == 0 {
			result.StopReason = steamdomain.StopExhausted
			return result, nil
		}
		for _, review := range page.Reviews {
			if !seenIDs[review.RecommendationID] {
				seenIDs[review.RecommendationID] = true
				result.Reviews = append(result.Reviews, review)
			}
		}
		if len(result.Reviews) >= options.MaxReviews {
			result.StopReason = steamdomain.StopReviewLimit
			return result, nil
		}
		if seenCursors[page.NextCursor] {
			result.StopReason = steamdomain.StopFailed
			return result, fmt.Errorf("collect Steam reviews after page %d: %w", result.PagesFetched, steamdomain.ErrCursorCycle)
		}
		seenCursors[page.NextCursor] = true
		cursor = page.NextCursor
	}
	result.StopReason = steamdomain.StopPageLimit
	return result, nil
}
