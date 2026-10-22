package reviewmcp

import (
	"errors"
	"testing"

	"github.com/shigaresso/player-lens/internal/steam"
)

func TestLimits(t *testing.T) {
	t.Run("既定値は分6回・24時間100回・同時2件", func(t *testing.T) {
		limits := DefaultLimits()
		if limits != (Limits{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 2}) {
			t.Fatal(limits)
		}
		if err := limits.Validate(); err != nil {
			t.Fatal(err)
		}
	})
	for _, tc := range []struct {
		name   string
		limits Limits
		valid  bool
	}{
		{"すべて1の下限を受け付ける", Limits{1, 1, 1}, true},
		{"日次が分次より小さくても独立した上限として扱う", Limits{10, 1, 2}, true},
		{"分次のゼロを拒否する", Limits{0, 100, 2}, false},
		{"分次の負数を拒否する", Limits{-1, 100, 2}, false},
		{"日次のゼロを拒否する", Limits{6, 0, 2}, false},
		{"日次の負数を拒否する", Limits{6, -1, 2}, false},
		{"同時数のゼロを拒否する", Limits{6, 100, 0}, false},
		{"同時数の負数を拒否する", Limits{6, 100, -1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.limits.Validate()
			if (err == nil) != tc.valid {
				t.Fatalf("limits=%+v err=%v", tc.limits, err)
			}
			if !tc.valid && !errors.Is(err, steam.ErrInvalidArgument) {
				t.Fatal(err)
			}
		})
	}
}

func TestNewHandlerRejectsInvalidLimits(t *testing.T) {
	handler, err := NewHandler(fetchFunc(nil), Limits{}, nil)
	if handler != nil || !errors.Is(err, steam.ErrInvalidArgument) {
		t.Fatalf("handler=%v err=%v", handler, err)
	}
}
