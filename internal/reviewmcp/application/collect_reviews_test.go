package application_test

import (
	"context"
	"testing"
	"time"

	mcpapplication "github.com/shigaresso/player-lens/internal/reviewmcp/application"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

type fetchFunc func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error)

func (f fetchFunc) FetchReviewsPage(ctx context.Context, id steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
	return f(ctx, id, options)
}

func TestCollectReviews(t *testing.T) {
	limits, err := mcpdomain.NewLimits(mcpdomain.LimitsConfig{RequestsPerMinute: 2, RequestsPerDay: 10, MaxConcurrent: 1})
	if err != nil {
		t.Fatal(err)
	}
	gate := mcpdomain.NewLimiter(limits, time.Now)
	var collectedContext context.Context
	fetcher := fetchFunc(func(ctx context.Context, id steamdomain.AppID, options steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		collectedContext = ctx
		if id != 440 || options.Language != "japanese" {
			t.Fatal("取得条件を伝えなかった")
		}
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 8*time.Second {
			t.Fatal("MCP の期限を設定しなかった")
		}
		return steamdomain.ReviewPage{}, nil
	})
	result := mcpapplication.CollectReviews(t.Context(), fetcher, 440, collectionOptions(t, steamdomain.CollectionOptionsConfig{Language: "japanese"}), gate)
	if result.Err != nil || result.Rejection != mcpdomain.NotRejected || result.Collection.AppID != 440 || result.Collection.StopReason != steamdomain.StopExhausted {
		t.Fatalf("result=%+v", result)
	}
	if collectedContext == nil || collectedContext.Err() != context.Canceled {
		t.Fatal("取得用 context を解放しなかった")
	}
	release, reason := gate.Acquire()
	if reason != mcpdomain.NotRejected {
		t.Fatalf("同時枠を返却しなかった: %v", reason)
	}
	release()
	if release, reason := gate.Acquire(); release != nil || reason != mcpdomain.MinuteLimitReached {
		t.Fatal("開始回数を戻した", reason)
	}
}

func collectionOptions(t *testing.T, config steamdomain.CollectionOptionsConfig) steamdomain.CollectionOptions {
	t.Helper()
	options, err := steamdomain.NewCollectionOptions(config)
	if err != nil {
		t.Fatal(err)
	}
	return options
}
