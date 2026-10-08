package steam

import (
	"context"
	"fmt"
)

// PageFetcher は1ページ取得の依存を表す。Client と同じ入力・応答の契約に従う。
// 通信と応答 Body の解放は実装側、接続プールの終了は組み立てた呼び出し側が担当する。
type PageFetcher interface {
	FetchReviewsPage(context.Context, uint32, PageOptions) (ReviewPage, error)
}

// StopReason は収集の終了理由を表す。上限到達は全件の取得完了を意味しない。
type StopReason string

const (
	// StopExhausted は空ページを取得し、対象条件の走査を終えたことを表す。
	StopExhausted StopReason = "exhausted"
	// StopReviewLimit はレビュー件数の上限で終了したことを表す。
	StopReviewLimit StopReason = "review_limit"
	// StopPageLimit はページ数の上限で終了したことを表す。
	StopPageLimit StopReason = "page_limit"
	// StopFailed は取得や収集の失敗を表す。詳細は返されたエラーで確認する。
	StopFailed StopReason = "failed"
)

// Collection は収集結果を表す。Reviews は同じ ID の最初の本文を取得順に保持する。
// PagesFetched は空ページを含む取得成功ページ数で、失敗したページは含まない。
type Collection struct {
	AppID        uint32
	Options      CollectionOptions
	Reviews      []Review
	PagesFetched int
	StopReason   StopReason
}

// CollectReviews は先頭から上限付きでページを取得する。再試行と保存は行わない。
// fetcher は有効な取得依存を渡す。入力不正ではゼロ値と ErrInvalidArgument を返す。
func CollectReviews(ctx context.Context, fetcher PageFetcher, appID uint32, options CollectionOptions) (Collection, error) {
	if fetcher == nil || appID == 0 {
		return Collection{}, fmt.Errorf("page fetcher and positive AppID are required: %w", ErrInvalidArgument)
	}
	options, err := NormalizeCollectionOptions(options)
	if err != nil {
		return Collection{}, fmt.Errorf("prepare Steam review collection: %w", err)
	}
	result := Collection{AppID: appID, Options: options, Reviews: []Review{}}
	seenIDs := make(map[string]bool)
	cursor := FirstPageCursor
	for result.PagesFetched < options.MaxPages {
		size := min(options.PageSize, options.MaxReviews-len(result.Reviews))
		page, err := fetcher.FetchReviewsPage(ctx, appID, PageOptions{Language: options.Language, Cursor: cursor, PageSize: size})
		if err != nil {
			return Collection{}, fmt.Errorf("collect Steam review page: %w", err)
		}
		result.PagesFetched++
		if len(page.Reviews) == 0 {
			result.StopReason = StopExhausted
			return result, nil
		}
		for _, review := range page.Reviews {
			if !seenIDs[review.ID] {
				seenIDs[review.ID] = true
				result.Reviews = append(result.Reviews, review)
			}
		}
		if len(result.Reviews) >= options.MaxReviews {
			result.StopReason = StopReviewLimit
			return result, nil
		}
		cursor = page.NextCursor
	}
	result.StopReason = StopPageLimit
	return result, nil
}
