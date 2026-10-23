package steam_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestCollectReviewsReviewType(t *testing.T) {
	for _, tt := range []struct {
		name string
		kind steam.ReviewType
		fail bool
	}{
		{"全評価を次のページへ維持する", steam.ReviewTypeAll, false},
		{"おすすめを次のページへ維持する", steam.ReviewTypePositive, false},
		{"非おすすめを次のページへ維持する", steam.ReviewTypeNegative, false},
		{"途中失敗でもおすすめの条件を保持する", steam.ReviewTypePositive, true},
		{"途中失敗でも非おすすめの条件を保持する", steam.ReviewTypeNegative, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			fetcher := pageFetcherFunc(func(_ context.Context, id steam.AppID, options steam.PageOptions) (steam.ReviewPage, error) {
				calls++
				cursor := steam.FirstPageCursor
				if calls == 2 {
					cursor = "next"
				}
				if id != 440 || options.ReviewType != tt.kind || options.Cursor != cursor || options.PageSize != 1 {
					t.Fatalf("page %d options=%+v", calls, options)
				}
				if calls == 2 && tt.fail {
					return steam.ReviewPage{}, io.ErrUnexpectedEOF
				}
				recommendationID := steam.RecommendationID(1)
				if calls == 2 {
					recommendationID = 2
				}
				return steam.ReviewPage{Reviews: []steam.Review{{RecommendationID: recommendationID, Recommended: tt.kind == steam.ReviewTypePositive}}, NextCursor: "next"}, nil
			})
			result, err := steam.CollectReviews(t.Context(), fetcher, 440, steam.CollectionOptions{ReviewType: tt.kind, PageSize: 1, MaxReviews: 2, MaxPages: 2})
			wantCount, wantReason := 2, steam.StopReviewLimit
			if tt.fail {
				wantCount, wantReason = 1, steam.StopFailed
			}
			if (err != nil) != tt.fail || (tt.fail && !errors.Is(err, io.ErrUnexpectedEOF)) || calls != 2 || result.Options.ReviewType != tt.kind || len(result.Reviews) != wantCount || result.PagesFetched != wantCount || result.StopReason != wantReason {
				t.Fatalf("result=%+v calls=%d err=%v", result, calls, err)
			}
		})
	}
}
