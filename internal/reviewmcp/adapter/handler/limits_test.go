package handler

import (
	"errors"
	"testing"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestNewHandlerRejectsInvalidLimits(t *testing.T) {
	handler, err := NewHandler(fetchFunc(nil), mcpdomain.Limits{}, nil)
	if handler != nil || !errors.Is(err, steamdomain.ErrInvalidArgument) {
		t.Fatalf("handler=%v err=%v", handler, err)
	}
}

func testLimits(t *testing.T, config mcpdomain.LimitsConfig) mcpdomain.Limits {
	t.Helper()
	limits, err := mcpdomain.NewLimits(config)
	if err != nil {
		t.Fatal(err)
	}

	return limits
}
