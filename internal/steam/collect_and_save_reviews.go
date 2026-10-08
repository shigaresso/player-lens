package steam

import (
	"context"
	"fmt"
)

// CollectionSaver は収集結果を保存する利用側の契約を表す。
// JSON・DB などの形式と資源の所有は実装側が担当し、context を尊重する。
// エラーは原因を保持して返し、再試行しない。原子的な保存は実装ごとに定める。
type CollectionSaver interface {
	SaveCollection(context.Context, Collection) error
}

// CollectAndSaveReviews は収集して結果を注入された保存先へ1回渡す。
// 有効な依存と非 nil の context を渡す。不正入力では取得・保存を行わない。
func CollectAndSaveReviews(ctx context.Context, fetcher PageFetcher, saver CollectionSaver, appID uint32, options CollectionOptions) (Collection, error) {
	if saver == nil {
		return Collection{}, fmt.Errorf("collection saver is required: %w", ErrInvalidArgument)
	}
	result, err := CollectReviews(ctx, fetcher, appID, options)
	if err != nil {
		return result, err
	}
	if err := saver.SaveCollection(ctx, result); err != nil {
		return result, fmt.Errorf("save Steam review collection: %w", err)
	}
	return result, nil
}
