package reviewmcp

import (
	"sync"
	"time"
)

// usageRejectionReason は利用制限の拒否理由を表す。ゼロ値は拒否なしを表す。
// 公開コードと説明文への変換は usageFailure が担当する。
type usageRejectionReason int

const (
	usageNotRejected usageRejectionReason = iota
	usageDailyLimitReached
	usageMinuteLimitReached
	usageConcurrentLimitReached
)

type window struct {
	start time.Time
	count int
}

func (w *window) reset(now time.Time, duration time.Duration) {
	if w.start.IsZero() || !now.Before(w.start.Add(duration)) {
		w.start = now
		w.count = 0
	}
}

type limiter struct {
	mu     sync.Mutex
	limits Limits
	now    func() time.Time
	minute window
	day    window
	active int
}

func newLimiter(limits Limits, now func() time.Time) *limiter {
	return &limiter{limits: limits, now: now}
}

func (l *limiter) acquire() (func(), usageRejectionReason) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.minute.reset(now, time.Minute)
	l.day.reset(now, 24*time.Hour)
	if l.day.count >= l.limits.RequestsPerDay {
		return nil, usageDailyLimitReached
	}
	if l.minute.count >= l.limits.RequestsPerMinute {
		return nil, usageMinuteLimitReached
	}
	if l.active >= l.limits.MaxConcurrent {
		return nil, usageConcurrentLimitReached
	}
	l.minute.count++
	l.day.count++
	l.active++
	return sync.OnceFunc(func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.active--
	}), usageNotRejected
}
