package reviewmcp

import (
	"errors"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestDefaultLimits(t *testing.T) {
	limits := DefaultLimits()
	if limits.Config() != (LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 2}) {
		t.Fatalf("既定値が変わった: %+v", limits.Config())
	}
}

func TestNewHandlerRejectsInvalidLimits(t *testing.T) {
	handler, err := NewHandler(fetchFunc(nil), Limits{}, nil)
	if handler != nil || !errors.Is(err, steam.ErrInvalidArgument) {
		t.Fatalf("handler=%v err=%v", handler, err)
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
