package domain

import (
	"testing"
)

func TestDefaultLimits(t *testing.T) {
	limits := DefaultLimits()
	if limits.Config() != (LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 2}) {
		t.Fatalf("既定値が変わった: %+v", limits.Config())
	}
}

func testLimits(t *testing.T, config LimitsConfig) Limits {
	t.Helper()
	limits, err := NewLimits(config)
	if err != nil {
		t.Fatal(err)
	}
	return limits
}
