package domain

import (
	"testing"
	"time"
)

func TestLimiterConcurrentAdmission(t *testing.T) {
	gate := NewLimiter(testLimits(t, LimitsConfig{RequestsPerMinute: 100, RequestsPerDay: 100, MaxConcurrent: 3}), time.Now)
	type admission struct {
		release func()
		reason  RejectionReason
	}
	results := make(chan admission, 80)
	start := make(chan struct{})
	for range 80 {
		go func() {
			<-start
			release, reason := gate.Acquire()
			results <- admission{release, reason}
		}()
	}
	close(start)
	var accepted []func()
	for range 80 {
		result := <-results
		if result.reason == NotRejected {
			accepted = append(accepted, result.release)
		} else if result.reason != ConcurrentLimitReached || result.release != nil {
			t.Fatalf("拒否=%+v", result)
		}
	}
	if len(accepted) != 3 {
		t.Fatalf("受付数=%d", len(accepted))
	}
	for _, release := range accepted {
		release()
	}
	release, reason := gate.Acquire()
	if reason != NotRejected {
		t.Fatal(reason)
	}
	release()
}

func TestLimiterBusyDoesNotConsumeBudget(t *testing.T) {
	gate := NewLimiter(testLimits(t, LimitsConfig{RequestsPerMinute: 4, RequestsPerDay: 10, MaxConcurrent: 2}), time.Now)
	first, reason := gate.Acquire()
	if reason != NotRejected {
		t.Fatal(reason)
	}
	second, reason := gate.Acquire()
	if reason != NotRejected {
		t.Fatal(reason)
	}
	if release, reason := gate.Acquire(); release != nil || reason != ConcurrentLimitReached {
		t.Fatalf("拒否=%v", reason)
	}
	first()
	first()
	third, reason := gate.Acquire()
	if reason != NotRejected {
		t.Fatal("busy が予算を消費した", reason)
	}
	if release, reason := gate.Acquire(); release != nil || reason != ConcurrentLimitReached {
		t.Fatalf("二重返却で枠が増えた: %v", reason)
	}
	second()
	third()
	fourth, reason := gate.Acquire()
	if reason != NotRejected {
		t.Fatal(reason)
	}
	defer fourth()
	if release, reason := gate.Acquire(); release != nil || reason != MinuteLimitReached {
		t.Fatalf("回数制限=%v", reason)
	}
}
