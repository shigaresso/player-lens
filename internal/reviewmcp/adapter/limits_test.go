package adapter

import (
	"errors"
	"testing"

	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/internal/testutil"
	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestNewHandlerRejectsInvalidLimits(t *testing.T) {
	handler, err := NewHandler(testutil.FetchFunc(nil), mcpdomain.Limits{}, localHTTPConfig(t))
	if handler != nil || !errors.Is(err, steamdomain.ErrInvalidArgument) {
		t.Fatalf("handler=%v err=%v", handler, err)
	}
}
