package domain

import (
	"fmt"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

// LimitsConfig は未検証の収集開始回数・同時実行数の設定値を表す。
// 各値は正の整数とし、利用する前に NewLimits へ渡す。
type LimitsConfig struct {
	RequestsPerMinute int
	RequestsPerDay    int
	MaxConcurrent     int
}

// Limits は生成時に正数を確認した、収集開始回数・同時実行数の上限を保持する。
// NewLimits または DefaultLimits で生成する。ゼロ値は未初期化であり、adapter.NewHandler は拒否する。
// 設定値は生成後に変更せず、利用回数や同時実行数の状態は handler が別に所有する。
type Limits struct {
	config LimitsConfig
}

// NewLimits はすべての上限が正であることを検査し、設定値のコピーを保持する。
// 不正ならゼロ値と ErrInvalidArgument を保持するエラーを返す。通信・利用枠の消費は行わない。
func NewLimits(config LimitsConfig) (Limits, error) {
	for _, item := range []struct {
		name  string
		value int
	}{
		{"分次", config.RequestsPerMinute},
		{"24時間", config.RequestsPerDay},
		{"同時数", config.MaxConcurrent},
	} {
		if item.value <= 0 {
			return Limits{}, fmt.Errorf("MCP の%sの上限は正の整数で指定する: %w", item.name, steamdomain.ErrInvalidArgument)
		}
	}
	return Limits{config: config}, nil
}

// DefaultLimits は分6回・24時間100回・同時2件の有効な初期値を返す。
func DefaultLimits() Limits {
	return Limits{config: LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 2}}
}

// Config は設定値のコピーを返す。コピーの変更は元の Limits へ影響しない。
// ゼロ値の Limits では、未初期化を表すゼロ値の設定を返す。
func (limits Limits) Config() LimitsConfig {
	return limits.config
}
