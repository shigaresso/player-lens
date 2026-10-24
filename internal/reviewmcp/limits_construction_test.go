package reviewmcp_test

import (
	"errors"
	"testing"

	"github.com/shigaresso/player-lens/internal/reviewmcp"
	"github.com/shigaresso/player-lens/internal/steam"
)

func TestNewLimits(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, tc := range []struct {
		name   string
		config reviewmcp.LimitsConfig
		valid  bool
	}{
		{"すべて1の下限を受け付ける", reviewmcp.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 1, MaxConcurrent: 1}, true},
		{"日次が分次より小さくても独立した上限として扱う", reviewmcp.LimitsConfig{RequestsPerMinute: 10, RequestsPerDay: 1, MaxConcurrent: 2}, true},
		{"Go API の正数には環境変数の31 bit 制約を適用しない", reviewmcp.LimitsConfig{RequestsPerMinute: maxInt, RequestsPerDay: maxInt, MaxConcurrent: maxInt}, true},
		{"分次のゼロを拒否する", reviewmcp.LimitsConfig{RequestsPerMinute: 0, RequestsPerDay: 100, MaxConcurrent: 2}, false},
		{"分次の負数を拒否する", reviewmcp.LimitsConfig{RequestsPerMinute: -1, RequestsPerDay: 100, MaxConcurrent: 2}, false},
		{"日次のゼロを拒否する", reviewmcp.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 0, MaxConcurrent: 2}, false},
		{"日次の負数を拒否する", reviewmcp.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: -1, MaxConcurrent: 2}, false},
		{"同時数のゼロを拒否する", reviewmcp.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 0}, false},
		{"同時数の負数を拒否する", reviewmcp.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: -1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits, err := reviewmcp.NewLimits(tc.config)
			if tc.valid {
				if err != nil || limits.Config() != tc.config {
					t.Fatalf("limits=%+v err=%v", limits, err)
				}
			} else if !errors.Is(err, steam.ErrInvalidArgument) || limits != (reviewmcp.Limits{}) {
				t.Fatalf("不正設定の生成結果: limits=%+v err=%v", limits, err)
			}
		})
	}
}
