package adapter

import (
	"fmt"
	"strconv"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
)

const (
	mcpRequestsPerMinuteEnv = "MCP_REQUESTS_PER_MINUTE"
	mcpRequestsPerDayEnv    = "MCP_REQUESTS_PER_DAY"
	mcpMaxConcurrentEnv     = "MCP_MAX_CONCURRENT"
)

// ReadMCPLimits は3つの環境設定を読み、解析と検査済みの上限の生成を行う。
// 各項目を分次・24時間・同時数の順に1回ずつ取得してから解析する。
// 未指定は既定値を使い、不正な設定は起動前に拒否する。
func ReadMCPLimits(getenv func(string) string) (mcpdomain.Limits, error) {
	input := mcpLimitsInput{
		requestsPerMinute: getenv(mcpRequestsPerMinuteEnv),
		requestsPerDay:    getenv(mcpRequestsPerDayEnv),
		maxConcurrent:     getenv(mcpMaxConcurrentEnv),
	}
	config, parseErr := parseMCPLimits(input)
	limits, err := mcpdomain.NewLimits(config)
	if parseErr != nil {
		return limits, parseErr
	}

	return limits, err
}

// mcpLimitsInput は環境変数から取得済みの未解析の文字列を保持する。
//
//contractcheck:data 環境変数から読み取った未検証の文字列入力
type mcpLimitsInput struct {
	requestsPerMinute string
	requestsPerDay    string
	maxConcurrent     string
}

// parseMCPLimits は取得済みの文字列に既定値を適用し、31 bit の整数へ変換する。
// I/O は行わず、正数条件の検査は NewLimits に委ねる。
func parseMCPLimits(input mcpLimitsInput) (mcpdomain.LimitsConfig, error) {
	config := mcpdomain.DefaultLimits().Config()
	for _, item := range []struct {
		key    string
		value  string
		target *int
	}{
		{key: mcpRequestsPerMinuteEnv, value: input.requestsPerMinute, target: &config.RequestsPerMinute},
		{key: mcpRequestsPerDayEnv, value: input.requestsPerDay, target: &config.RequestsPerDay},
		{key: mcpMaxConcurrentEnv, value: input.maxConcurrent, target: &config.MaxConcurrent},
	} {
		if item.value == "" {
			continue
		}

		number, err := strconv.ParseUint(item.value, 10, 31)
		if err != nil {
			return mcpdomain.LimitsConfig{}, fmt.Errorf("%s は正の整数で指定する: %w", item.key, err)
		}

		*item.target = int(number)
	}

	return config, nil
}
