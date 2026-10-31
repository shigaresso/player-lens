package adapter

import (
	"fmt"
	"io"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// readReviewPageBody は本文を上限付きで読み取り、内部のページへ変換する。
// 呼び出し側は HTTP 200 を確認した本文、正の App ID・正規化済みの条件を渡す。
// ステータスの判断、Body の Close、context の寿命は呼び出し側が担当する。
func readReviewPageBody(reader io.Reader, appID steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
	body, err := io.ReadAll(io.LimitReader(reader, maxResponseBytes+1))
	if err != nil {
		return steamdomain.ReviewPage{}, fmt.Errorf("read Steam review response: %w", err)
	}

	if len(body) > maxResponseBytes {
		return steamdomain.ReviewPage{}, ErrResponseTooLarge
	}

	return decodeReviewPage(appID, options, body)
}
