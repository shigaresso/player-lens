package application_test

import (
	"context"
	"errors"
	"io"
	"testing"

	steamapplication "github.com/shigaresso/player-lens/internal/steam/application"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectReviewsReviewType(t *testing.T) {
	for _, tt := range []struct {
		name string
		kind steamdomain.ReviewType
		fail bool
	}{
		{"全評価を次のページへ維持する", steamdomain.ReviewTypeAll, false},
		{"おすすめを次のページへ維持する", steamdomain.ReviewTypePositive, false},
		{"非おすすめを次のページへ維持する", steamdomain.ReviewTypeNegative, false},
		{"途中失敗でもおすすめの条件を保持する", steamdomain.ReviewTypePositive, true},
		{"途中失敗でも非おすすめの条件を保持する", steamdomain.ReviewTypeNegative, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			fetcher := pageFetcherFunc(func(_ context.Context, id steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
				calls++
				cursor := steamdomain.FirstPageCursor
				if calls == 2 {
					cursor = "next"
				}
				if id != 440 || options.ReviewType != tt.kind || options.Cursor != cursor || options.PageSize != 1 {
					t.Fatalf("page %d options=%+v", calls, options)
				}
				if calls == 2 && tt.fail {
					return steamdomain.ReviewPage{}, io.ErrUnexpectedEOF
				}
				recommendationID := steamdomain.RecommendationID(1)
				if calls == 2 {
					recommendationID = 2
				}
				return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: recommendationID, Recommended: tt.kind == steamdomain.ReviewTypePositive}}, NextCursor: "next"}, nil
			})
			result, err := steamapplication.CollectReviews(t.Context(), fetcher, 440, steamdomain.CollectionOptions{ReviewType: tt.kind, PageSize: 1, MaxReviews: 2, MaxPages: 2})
			wantCount, wantReason := 2, steamdomain.StopReviewLimit
			if tt.fail {
				wantCount, wantReason = 1, steamdomain.StopFailed
			}
			if (err != nil) != tt.fail || (tt.fail && !errors.Is(err, io.ErrUnexpectedEOF)) || calls != 2 || result.Options.ReviewType != tt.kind || len(result.Reviews) != wantCount || result.PagesFetched != wantCount || result.StopReason != wantReason {
				t.Fatalf("result=%+v calls=%d err=%v", result, calls, err)
			}
		})
	}
}
