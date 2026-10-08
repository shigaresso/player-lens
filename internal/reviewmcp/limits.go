package reviewmcp

import (
	"fmt"

	"github.com/shigaresso/player-lens/internal/steam"
)

// Limits は1プロセス全体の収集開始回数と同時実行数の上限を表す。
// 各値は正の整数とし、0による無制限化は許可しない。
type Limits struct {
	RequestsPerMinute int
	RequestsPerDay    int
	MaxConcurrent     int
}

// DefaultLimits は1分6回・24時間100回・同時2件の初期値を返す。
func DefaultLimits() Limits {
	return Limits{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 2}
}

// Validate はすべての上限が正であることを確認し、不正なら ErrInvalidArgument を保持して返す。
func (limits Limits) Validate() error {
	for _, item := range []struct {
		name  string
		value int
	}{
		{"分次", limits.RequestsPerMinute},
		{"24時間", limits.RequestsPerDay},
		{"同時数", limits.MaxConcurrent},
	} {
		if item.value <= 0 {
			return fmt.Errorf("MCP の%sの上限は正の整数で指定する: %w", item.name, steam.ErrInvalidArgument)
		}
	}
	return nil
}
