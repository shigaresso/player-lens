package domain

import (
	"sync"
	"time"
)

// RejectionReason は利用制限の拒否理由を表す。ゼロ値は拒否なしを表す。
// 公開コードと説明文への変換は adapter が担当する。
type RejectionReason int

const (
	// NotRejected は受付に成功したことを表す。
	NotRejected RejectionReason = iota
	// DailyLimitReached は24時間の開始回数を使い切ったことを表す。
	DailyLimitReached
	// MinuteLimitReached は分次の開始回数を使い切ったことを表す。
	MinuteLimitReached
	// ConcurrentLimitReached は同時実行枠を使い切ったことを表す。
	ConcurrentLimitReached
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

// Limiter は共有する開始回数と同時実行数の状態を所有する。
// ゼロ値は使用せず、NewLimiter で生成する。
type Limiter struct {
	mu     sync.Mutex
	limits Limits
	now    func() time.Time
	minute window
	day    window
	active int
}

// NewLimiter は生成済みの有効な上限と非 nil の時計で空の受付状態を作る。
// 同じ handler 内で状態を共有する。
func NewLimiter(limits Limits, now func() time.Time) *Limiter {
	return &Limiter{limits: limits, now: now}
}

// Acquire は開始回数と同時枠をまとめて確保し、拒否時は nil と理由を返す。
// 成功時は NotRejected と返却関数を返す。呼び出し側は必ず返却する。
// 並行呼び出しと二重返却に対応し、開始回数は戻さない。
func (l *Limiter) Acquire() (func(), RejectionReason) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.minute.reset(now, time.Minute)
	l.day.reset(now, 24*time.Hour)
	if l.day.count >= l.limits.config.RequestsPerDay {
		return nil, DailyLimitReached
	}

	if l.minute.count >= l.limits.config.RequestsPerMinute {
		return nil, MinuteLimitReached
	}

	if l.active >= l.limits.config.MaxConcurrent {
		return nil, ConcurrentLimitReached
	}

	l.minute.count++
	l.day.count++
	l.active++
	return sync.OnceFunc(func() {
		l.mu.Lock()
		defer l.mu.Unlock()

		l.active--
	}), NotRejected
}
