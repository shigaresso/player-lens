package steam_test

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestCollectAndSaveReviewsFailure(t *testing.T) {
	for _, test := range []struct {
		name    string
		failAt  int
		saveErr error
	}{
		{"最初の取得失敗も条件と空の結果を保存する", 1, nil},
		{"途中失敗の取得済み結果を保存する", 2, nil},
		{"収集と保存の両方の原因を保持する", 2, io.ErrClosedPipe},
		{"保存だけ失敗しても収集結果を保持する", 0, io.ErrClosedPipe},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls, saves := 0, 0
			var saved steam.Collection
			fetcher := pageFetcherFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
				calls++
				if calls == test.failAt {
					return steam.ReviewPage{}, io.ErrUnexpectedEOF
				}
				if calls == 1 {
					return steam.ReviewPage{Reviews: collectionReviews(1), NextCursor: "next"}, nil
				}
				return steam.ReviewPage{}, nil
			})
			saver := collectionSaverFunc(func(_ context.Context, result steam.Collection) error {
				saves++
				saved = result
				return test.saveErr
			})
			result, err := steam.CollectAndSaveReviews(t.Context(), fetcher, saver, 440, steam.CollectionOptions{})
			if saves != 1 || !reflect.DeepEqual(saved, result) || result.AppID != 440 {
				t.Fatalf("result=%+v saved=%+v saves=%d err=%v", result, saved, saves, err)
			}
			if test.failAt > 0 {
				if !errors.Is(err, io.ErrUnexpectedEOF) || calls != test.failAt || result.PagesFetched != test.failAt-1 || result.StopReason != steam.StopFailed {
					t.Fatalf("取得失敗と部分結果: result=%+v calls=%d err=%v", result, calls, err)
				}
			} else if result.StopReason != steam.StopExhausted {
				t.Fatalf("保存失敗で収集の終了理由を変えた: %+v", result)
			}
			if test.saveErr != nil && !errors.Is(err, test.saveErr) {
				t.Fatalf("保存の原因を失った: %v", err)
			}
		})
	}
}
