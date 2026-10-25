package adapter

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// conversionProbeError は公開用の失敗分類が原因を調べる時点を観測する。
// Error は収集時のラップでも呼ばれるため、errors.Is による分類を使う。
type conversionProbeError struct{ classify func() }

func (*conversionProbeError) Error() string { return "fetch failed" }

func (e *conversionProbeError) Is(target error) bool {
	if target == context.Canceled {
		e.classify()
	}
	return false
}

func TestCollectKeepsSlotDuringFailureConversion(t *testing.T) {
	gate := mcpdomain.NewLimiter(testLimits(t, mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 10, MaxConcurrent: 1}), time.Now)
	classifications := 0
	cause := &conversionProbeError{classify: func() {
		classifications++
		release, reason := gate.Acquire()
		if release != nil {
			release()
		}
		if reason != mcpdomain.ConcurrentLimitReached {
			t.Errorf("応答変換中に同時枠が空いている: reason=%v", reason)
		}
	}}
	result, err := collect(t.Context(), fetchFunc(func(context.Context, steamdomain.AppID, steamdomain.PageOptions) (steamdomain.ReviewPage, error) {
		return steamdomain.ReviewPage{}, cause
	}), json.RawMessage(`{"app_id":440}`), gate)
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if classifications == 0 || decodeOutput(t, result).ErrorCode != "fetch_failed" {
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
