package adapter_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	reviewcli "github.com/shigaresso/player-lens/internal/reviewcli/adapter"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestRunReviewType(t *testing.T) {
	for _, tt := range []struct {
		name    string
		args    []string
		want    steamdomain.ReviewType
		invalid bool
	}{
		{"省略は全評価", nil, steamdomain.ReviewTypeAll, false},
		{"空指定は全評価", []string{"--review-type="}, steamdomain.ReviewTypeAll, false},
		{"全評価を指定する", []string{"--review-type=all"}, steamdomain.ReviewTypeAll, false},
		{"おすすめを指定する", []string{"--review-type", "positive"}, steamdomain.ReviewTypePositive, false},
		{"非おすすめを指定する", []string{"--review-type=negative"}, steamdomain.ReviewTypeNegative, false},
		{"未知値は通信と保存前に拒否する", []string{"--review-type=mixed"}, 0, true},
		{"大文字は拒否する", []string{"--review-type=Positive"}, 0, true},
		{"前後空白は拒否する", []string{"--review-type= negative "}, 0, true},
		{"値の欠落は拒否する", []string{"--review-type"}, 0, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls, saves := 0, 0
			fetcher := fetcherFunc(func(_ context.Context, id steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				calls++
				if tt.invalid || id != 440 || options.ReviewType != tt.want {
					t.Fatalf("取得条件=%+v", options)
				}

				return steamdomain.ReviewPage{}, nil
			})
			saver := saverFunc(func(_ context.Context, result steamdomain.Collection) error {
				saves++
				if tt.invalid || result.Options.ReviewType != tt.want || result.StopReason != steamdomain.StopExhausted {
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
