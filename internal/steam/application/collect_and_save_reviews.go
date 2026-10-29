package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// CollectionSaver は収集結果を保存する利用側の契約を表す。
// JSON・DB などの形式と資源の所有は実装側が担当し、context を尊重する。
// エラーは原因を保持して返し、再試行しない。原子的な保存は実装ごとに定める。
type CollectionSaver interface {
	SaveCollection(context.Context, steamdomain.Collection) error
}

// CollectAndSaveReviews は収集して結果を注入された保存先へ1回渡す。
// 有効な依存と非 nil の context を渡す。不正入力では取得・保存を行わない。
// 途中失敗でも部分結果を保存し、収集と保存の両方の原因を保持する。
// 保存は親の値を維持し、キャンセル・期限から独立した最大5秒の context を渡す。
// 保存実装が期限を尊重する責務を持ち、操作が派生 context を解放する。
func CollectAndSaveReviews(ctx context.Context, fetcher PageFetcher, saver CollectionSaver, appID steamdomain.AppID, options steamdomain.CollectionOptions) (steamdomain.Collection, error) {
	if saver == nil {
		return steamdomain.Collection{}, fmt.Errorf("collection saver is required: %w", steamdomain.ErrInvalidArgument)
	}

	result, collectErr := CollectReviews(ctx, fetcher, appID, options)
	if result.AppID == 0 {
		return result, collectErr
	}

	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := saver.SaveCollection(saveCtx, result); err != nil {
		return result, errors.Join(collectErr, fmt.Errorf("save Steam review collection: %w", err))
	}

	return result, collectErr
}
