package reviewmcp

import (
	"testing"
	"time"
)

func TestLimiterWindows(t *testing.T) {
	for _, tc := range []struct {
		name    string
		limits  Limits
		offsets []time.Duration
		codes   []string
	}{
		{"分次の上限と直前・ちょうど・直後の境界", Limits{RequestsPerMinute: 1, RequestsPerDay: 10, MaxConcurrent: 1}, []time.Duration{0, time.Minute - time.Nanosecond, time.Minute, time.Minute + time.Nanosecond}, []string{"", "rate_limited", "", "rate_limited"}},
		{"24時間の上限と直前・ちょうど・直後の境界", Limits{RequestsPerMinute: 10, RequestsPerDay: 1, MaxConcurrent: 1}, []time.Duration{0, 24*time.Hour - time.Nanosecond, 24 * time.Hour, 24*time.Hour + time.Nanosecond}, []string{"", "daily_limit_reached", "", "daily_limit_reached"}},
		{"分次での拒否は日次回数を消費しない", Limits{RequestsPerMinute: 1, RequestsPerDay: 2, MaxConcurrent: 1}, []time.Duration{0, 0, time.Minute, 2 * time.Minute}, []string{"", "rate_limited", "", "daily_limit_reached"}},
		{"日次は分次より先に判定する", Limits{RequestsPerMinute: 1, RequestsPerDay: 1, MaxConcurrent: 1}, []time.Duration{0, 0}, []string{"", "daily_limit_reached"}},
		{"日次の更新でも未終了の分次窓は保つ", Limits{RequestsPerMinute: 1, RequestsPerDay: 2, MaxConcurrent: 1}, []time.Duration{0, 24*time.Hour - 30*time.Second, 24 * time.Hour, 24*time.Hour + 30*time.Second}, []string{"", "", "rate_limited", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
			now := start
			gate := newLimiter(tc.limits, func() time.Time { return now })
			// 構築時ではなく最初の受付時から窓が始まる。
			start = start.Add(7 * time.Hour)
			for i, offset := range tc.offsets {
				now = start.Add(offset)
				if code := gate.allow(); code != tc.codes[i] {
					t.Fatalf("受付%d 時刻=%v code=%q want=%q", i, offset, code, tc.codes[i])
				}
			}
		})
	}
}
