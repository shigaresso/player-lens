package domain

import (
	"testing"
	"time"
)

func TestLimiterWindows(t *testing.T) {
	for _, tc := range []struct {
		name    string
		limits  Limits
		offsets []time.Duration
		reasons []RejectionReason
	}{
		{"分次の上限と直前・ちょうど・直後の境界", testLimits(t, LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 10, MaxConcurrent: 1}), []time.Duration{0, time.Minute - time.Nanosecond, time.Minute, time.Minute + time.Nanosecond}, []RejectionReason{NotRejected, MinuteLimitReached, NotRejected, MinuteLimitReached}},
		{"24時間の上限と直前・ちょうど・直後の境界", testLimits(t, LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 1, MaxConcurrent: 1}), []time.Duration{0, 24*time.Hour - time.Nanosecond, 24 * time.Hour, 24*time.Hour + time.Nanosecond}, []RejectionReason{NotRejected, DailyLimitReached, NotRejected, DailyLimitReached}},
		{"分次での拒否は日次回数を消費しない", testLimits(t, LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 2, MaxConcurrent: 1}), []time.Duration{0, 0, time.Minute, 2 * time.Minute}, []RejectionReason{NotRejected, MinuteLimitReached, NotRejected, DailyLimitReached}},
		{"日次は分次より先に判定する", testLimits(t, LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 1, MaxConcurrent: 1}), []time.Duration{0, 0}, []RejectionReason{NotRejected, DailyLimitReached}},
		{"日次の更新でも未終了の分次窓は保つ", testLimits(t, LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 2, MaxConcurrent: 1}), []time.Duration{0, 24*time.Hour - 30*time.Second, 24 * time.Hour, 24*time.Hour + 30*time.Second}, []RejectionReason{NotRejected, NotRejected, MinuteLimitReached, NotRejected}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
			now := start
			gate := NewLimiter(tc.limits, func() time.Time { return now })
			// 構築時ではなく最初の受付時から窓が始まる。
			start = start.Add(7 * time.Hour)
			for i, offset := range tc.offsets {
				now = start.Add(offset)
				release, reason := gate.Acquire()
				if reason != tc.reasons[i] {
					t.Fatalf("受付%d 時刻=%v reason=%v want=%v", i, offset, reason, tc.reasons[i])
				}

				if release != nil {
					release()
				}
			}
		})
	}
}
