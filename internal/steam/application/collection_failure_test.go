package application_test

import (
	"context"
	"errors"
	"io"
	"reflect"
	"testing"

	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectReviewsFailure(t *testing.T) {
	cases := []struct {
		name       string
		failOn     int
		cause      error
		otherCause error
	}{
		{"先頭ページの失敗は空の部分結果を返す", 1, io.ErrUnexpectedEOF, nil},
		{"途中の読み取り失敗は成功したページだけ保持する", 2, io.ErrUnexpectedEOF, nil},
		{"HTTP429を保持し再試行しない", 2, &steamadapter.HTTPStatusError{StatusCode: 429}, nil},
		{"キャンセルの原因を保持する", 2, context.Canceled, nil},
		{"期限切れの原因を保持する", 2, context.DeadlineExceeded, nil},
		{"複数の原因を保持する", 2, io.ErrUnexpectedEOF, io.ErrClosedPipe},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			fetcher := pageFetcherFunc(func(_ context.Context, _ steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				calls++
				if calls > tt.failOn {
					t.Fatal("失敗後に再取得した")
				}
				if calls == tt.failOn {
					return steamdomain.ReviewPage{}, errors.Join(tt.cause, tt.otherCause)
				}
				return steamdomain.ReviewPage{AppID: 440, Options: options, Reviews: collectionReviews(1), NextCursor: "A"}, nil
			})
			options := steamdomain.CollectionOptionsConfig{PageSize: 1, MaxReviews: 3, MaxPages: 4}
			got, err := steamapplication.CollectReviews(context.Background(), fetcher, 440, collectionOptions(t, options))
			want := []steamdomain.Review{}
			if tt.failOn > 1 {
				want = collectionReviews(1)
			}
			if calls != tt.failOn || got.AppID != 440 || got.Options.Language != "english" || got.Options.PageSize != 1 || got.Options.MaxReviews != 3 || got.Options.MaxPages != 4 || got.PagesFetched != tt.failOn-1 || got.StopReason != steamdomain.StopFailed || !reflect.DeepEqual(got.Reviews, want) || !errors.Is(err, tt.cause) {
				t.Fatalf("result=%#v calls=%d error=%v", got, calls, err)
			}
			if tt.otherCause != nil && !errors.Is(err, tt.otherCause) {
				t.Fatalf("追加の原因を失った: %v", err)
			}
			var status *steamadapter.HTTPStatusError
			if errors.As(tt.cause, &status) {
				var gotStatus *steamadapter.HTTPStatusError
				if !errors.As(err, &gotStatus) || gotStatus.StatusCode != status.StatusCode {
					t.Fatalf("ステータスを失った: %v", err)
				}
			}
		})
	}
}
