package domain_test

import (
	"errors"
	"testing"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestNewLimits(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, tc := range []struct {
		name   string
		config mcpdomain.LimitsConfig
		valid  bool
	}{
		{"すべて1の下限を受け付ける", mcpdomain.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 1, MaxConcurrent: 1}, true},
		{"日次が分次より小さくても独立した上限として扱う", mcpdomain.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 1, MaxConcurrent: 2}, true},
		{"Go API の正数には環境変数の31 bit 制約を適用しない", mcpdomain.LimitsConfig{RequestsPerMinute: maxInt, RequestsPerDay: maxInt, MaxConcurrent: maxInt}, true},
		{"分次のゼロを拒否する", mcpdomain.LimitsConfig{RequestsPerMinute: 0, RequestsPerDay: 100, MaxConcurrent: 2}, false},
		{"分次の負数を拒否する", mcpdomain.LimitsConfig{RequestsPerMinute: -1, RequestsPerDay: 100, MaxConcurrent: 2}, false},
		{"日次のゼロを拒否する", mcpdomain.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 0, MaxConcurrent: 2}, false},
		{"日次の負数を拒否する", mcpdomain.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: -1, MaxConcurrent: 2}, false},
		{"同時数のゼロを拒否する", mcpdomain.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 0}, false},
		{"同時数の負数を拒否する", mcpdomain.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: -1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits, err := mcpdomain.NewLimits(tc.config)
			if tc.valid {
				if err != nil || limits.Config() != tc.config {
					t.Fatalf("limits=%+v err=%v", limits, err)
				}
			} else if !errors.Is(err, steamdomain.ErrInvalidArgument) || limits != (mcpdomain.Limits{}) {
				t.Fatalf("不正設定の生成結果: limits=%+v err=%v", limits, err)
			}
		})
	}
}

func TestLimitsConfigIsolation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		changeCopy bool
	}{
		{"生成元の設定の変更を引き継がない", false},
		{"取得した設定のコピーの変更を引き継がない", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := mcpdomain.LimitsConfig{RequestsPerMinute: 3, RequestsPerDay: 20, MaxConcurrent: 1}
			config := want
			limits, err := mcpdomain.NewLimits(config)
			if err != nil {
				t.Fatal(err)
			}

			if tc.changeCopy {
				config = limits.Config()
			}

			config.RequestsPerMinute = 0
			config.RequestsPerDay = 0
			config.MaxConcurrent = 0
			if got := limits.Config(); got != want {
				t.Fatalf("生成済みの上限が変更された: %+v; want %+v", got, want)
			}
		})
	}

	t.Run("未初期化の値からはゼロ設定を返す", func(t *testing.T) {
		var limits mcpdomain.Limits
		if config := limits.Config(); config != (mcpdomain.LimitsConfig{}) {
			t.Fatalf("未初期化の設定 = %+v", config)
		}
	})
}
