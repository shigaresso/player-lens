package reviewcli_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/shigaresso/player-lens/internal/reviewcli"
	"github.com/shigaresso/player-lens/internal/steam"
)

func TestRunReviewType(t *testing.T) {
	for _, tt := range []struct {
		name    string
		args    []string
		want    steam.EUserReviewsReviewType
		invalid bool
	}{
		{"省略は全評価", nil, steam.ReviewTypeAll, false},
		{"空指定は全評価", []string{"--review-type="}, steam.ReviewTypeAll, false},
		{"全評価を指定する", []string{"--review-type=all"}, steam.ReviewTypeAll, false},
		{"おすすめを指定する", []string{"--review-type", "positive"}, steam.ReviewTypePositive, false},
		{"非おすすめを指定する", []string{"--review-type=negative"}, steam.ReviewTypeNegative, false},
		{"未知値は通信と保存前に拒否する", []string{"--review-type=mixed"}, 0, true},
		{"大文字は拒否する", []string{"--review-type=Positive"}, 0, true},
		{"前後空白は拒否する", []string{"--review-type= negative "}, 0, true},
		{"値の欠落は拒否する", []string{"--review-type"}, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls, saves := 0, 0
			fetcher := fetcherFunc(func(_ context.Context, id steam.AppID, options steam.PageOptions) (steam.ReviewPage, error) {
				calls++
				if tt.invalid || id != 440 || options.ReviewType != tt.want {
					t.Fatalf("取得条件=%+v", options)
				}
				return steam.ReviewPage{}, nil
			})
			saver := saverFunc(func(_ context.Context, result steam.Collection) error {
				saves++
				if tt.invalid || result.Options.ReviewType != tt.want || result.StopReason != steam.StopExhausted {
					t.Fatalf("保存条件=%+v", result)
				}
				return nil
			})
			args := append([]string{"--app-id=440", "--max-reviews=1", "--max-pages=1"}, tt.args...)
			err := reviewcli.Run(t.Context(), args, fetcher, saver)
			if tt.invalid {
				if !errors.Is(err, reviewcli.ErrUsage) || calls != 0 || saves != 0 {
					t.Fatalf("calls=%d saves=%d err=%v", calls, saves, err)
				}
			} else if err != nil || calls != 1 || saves != 1 {
				t.Fatalf("calls=%d saves=%d err=%v", calls, saves, err)
			}
		})
	}
	if !strings.Contains(reviewcli.Usage, "--review-type") || !strings.Contains(reviewcli.Usage, "positive") || !strings.Contains(reviewcli.Usage, "negative") {
		t.Fatal("評価種別の使用方法がない")
	}
}
