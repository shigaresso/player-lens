package reviewmcp

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestHandlerLimits(t *testing.T) {
	handler, err := NewHandler(fetchFunc(func(context.Context, uint32, steam.PageOptions) (steam.ReviewPage, error) {
		t.Fatal("拒否する HTTP リクエストで取得した")
		return steam.ReviewPage{}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name        string
		origin      string
		knownLength bool
		status      int
	}{
		{"既知の本文長でも16KiB超過を拒否する", "", true, 413},
		{"長さ未指定でも16KiB超過を拒否する", "", false, 413},
		{"同じOriginでも本文の上限を維持する", "http://example.com", true, 413},
		{"異なるOriginからの呼び出しを拒否する", "https://external.example", true, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "http://example.com/mcp", strings.NewReader(strings.Repeat("x", 16*1024+1)))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Accept", "application/json, text/event-stream")
			if !tc.knownLength {
				r.ContentLength = -1
			}
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status=%d want=%d", w.Code, tc.status)
			}
		})
	}
}

func TestNewHandlerRejectsNilFetcher(t *testing.T) {
	handler, err := NewHandler(nil)
	if handler != nil || !errors.Is(err, steam.ErrInvalidArgument) {
		t.Fatalf("handler=%v err=%v", handler, err)
	}
}
