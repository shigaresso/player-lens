package handler

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectionReviewTypePartialFailure(t *testing.T) {
	calls := 0
	fetcher := testutil.FetchFunc(func(_ context.Context, _ steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		calls++
		if options.ReviewType != steamdomain.ReviewTypeNegative {
			t.Fatalf("options=%+v", options)
		}

		if calls == 2 {
			return steamdomain.ReviewPage{}, io.ErrUnexpectedEOF
		}

		return steamdomain.ReviewPage{Reviews: []steamdomain.Review{{RecommendationID: 1, Text: "取得済み"}}, NextCursor: "next"}, nil
	})
	result, err := collect(t.Context(), fetcher, json.RawMessage(`{"app_id":440,"review_type":"negative","page_size":1,"max_reviews":2,"max_pages":2}`), mcpdomain.NewLimiter(mcpdomain.DefaultLimits(), time.Now))
	if err != nil {
		t.Fatal(err)
	}

	out := testutil.DecodeOutput(t, result)
	if !result.IsError || out.ErrorCode != "fetch_failed" || out.ReturnedCount != 1 || out.Collection.Options.ReviewType != "negative" || out.Collection.PagesFetched != 1 || out.Collection.StopReason != steamdomain.StopFailed || calls != 2 {
		t.Fatalf("out=%+v calls=%d", out, calls)
	}
}
