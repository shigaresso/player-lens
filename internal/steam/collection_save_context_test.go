package steam_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestCollectAndSaveReviewsSaveContext(t *testing.T) {
	type traceKey struct{}
	for _, test := range []struct {
		name     string
		canceled bool
		expired  bool
		saveErr  error
	}{
		{"正常終了でも保存期限と解放を管理する", false, false, nil},
		{"収集キャンセル後も部分結果を保存する", true, false, nil},
		{"収集期限切れ後も部分結果を保存する", false, true, nil},
		{"保存失敗でも派生contextを解放する", false, false, io.ErrClosedPipe},
	} {
		t.Run(test.name, func(t *testing.T) {
			parent := context.WithValue(t.Context(), traceKey{}, "trace")
			if test.expired {
				var cancel context.CancelFunc
				parent, cancel = context.WithDeadline(parent, time.Now().Add(-time.Second))
				t.Cleanup(cancel)
			} else {
				var cancel context.CancelFunc
				parent, cancel = context.WithCancel(parent)
				t.Cleanup(cancel)
				if test.canceled {
					cancel()
				}
			}
			before := parent.Err()
			var captured context.Context
			fetcher := pageFetcherFunc(func(context.Context, steam.AppID, steam.PageOptions) (steam.ReviewPage, error) {
				return steam.ReviewPage{}, nil
			})
			saver := collectionSaverFunc(func(ctx context.Context, result steam.Collection) error {
				captured = ctx
				deadline, ok := ctx.Deadline()
				remaining := time.Until(deadline)
				if ctx.Err() != nil || ctx.Value(traceKey{}) != "trace" || !ok || remaining <= 4*time.Second || remaining > 5*time.Second {
					t.Errorf("保存用context: err=%v value=%v remaining=%v deadline=%v", ctx.Err(), ctx.Value(traceKey{}), remaining, ok)
				}
				return test.saveErr
			})
			_, err := steam.CollectAndSaveReviews(parent, fetcher, saver, 440, steam.CollectionOptions{})
			if captured == nil || !errors.Is(captured.Err(), context.Canceled) || parent.Err() != before {
				t.Fatalf("派生contextの解放または親の保持: captured=%v parent=%v", captured, parent.Err())
			}
			if before != nil && !errors.Is(err, before) {
				t.Fatalf("収集の失敗を失った: %v", err)
			}
			if test.saveErr != nil && !errors.Is(err, test.saveErr) {
				t.Fatalf("保存の失敗を失った: %v", err)
			}
		})
	}
}
