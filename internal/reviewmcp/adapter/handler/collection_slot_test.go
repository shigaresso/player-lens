package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestCollectReleasesSlotBeforeFailureConversion(t *testing.T) {
	gate := mcpdomain.NewLimiter(testutil.Limits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 10, MaxConcurrent: 1}), time.Now)
	classifications := 0
	cause := &testutil.ConversionProbeError{Classify: func() {
		classifications++
		release, reason := gate.Acquire()
		if release != nil {
			release()
		}

		if reason != mcpdomain.NotRejected {
			t.Errorf("応答変換前に同時枠を返却しなかった: reason=%v", reason)
		}
	}}
	result, err := callCollectReviews(t, t.Context(), testutil.FetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		return steamdomain.ReviewPage{}, cause
	}), json.RawMessage(`{"app_id":440}`), gate)
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("result=%+v err=%v", result, err)
	}

	if classifications == 0 || testutil.DecodeOutput(t, result).ErrorCode != "fetch_failed" {
		t.Fatal("実際の応答変換で取得の失敗を分類しなかった")
	}

	release, reason := gate.Acquire()
	if release != nil {
		t.Cleanup(release)
	}

	if reason != mcpdomain.NotRejected {
		t.Fatalf("応答変換後に同時枠を返却しなかった: %v", reason)
	}
}
