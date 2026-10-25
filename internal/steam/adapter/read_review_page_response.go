package adapter

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// readReviewPageResponse は有効な HTTP 応答の読み取り・解釈・Body の Close を担当する。
// 呼び出し側は Do が成功した応答と、正の App ID・正規化済みの条件を渡す。
// 成功・失敗・読み取り中の panic とも Body を1回閉じ、Close は解釈後に行う。
// Close 失敗ではページをゼロ値にして元の原因と併せて返し、panic は伝播する。
func readReviewPageResponse(response *http.Response, appID steamdomain.AppID, options steamdomain.PageOptions) (page steamdomain.ReviewPage, err error) {
	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			page = steamdomain.ReviewPage{}
			err = errors.Join(err, fmt.Errorf("close Steam review response: %w", closeErr))
		}
	}()

	if response.StatusCode != http.StatusOK {
		return steamdomain.ReviewPage{}, &HTTPStatusError{StatusCode: response.StatusCode}
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return steamdomain.ReviewPage{}, fmt.Errorf("read Steam review response: %w", err)
	}

	if len(body) > maxResponseBytes {
		return steamdomain.ReviewPage{}, ErrResponseTooLarge
	}

	return decodeReviewPage(appID, options, body)
}
