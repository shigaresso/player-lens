package reviewmcp

import (
	"sync"
	"time"
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
}

func newLimiter(limits Limits, now func() time.Time) *limiter {
	return &limiter{limits: limits, now: now}
}

func (l *limiter) allow() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.minute.reset(now, time.Minute)
	l.day.reset(now, 24*time.Hour)
	if l.day.count >= l.limits.RequestsPerDay {
		return "daily_limit_reached"
	}
	if l.minute.count >= l.limits.RequestsPerMinute {
		return "rate_limited"
	}
	l.minute.count++
	l.day.count++
	return ""
}
