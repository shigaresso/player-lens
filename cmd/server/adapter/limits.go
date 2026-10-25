package adapter

import (
	"fmt"
	"strconv"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
)

// ReadMCPLimits は環境設定を31 bit の整数として解析し、検査済みの上限を返す。
// 未指定は既定値を使い、不正な設定は起動前に拒否する。
func ReadMCPLimits(getenv func(string) string) (mcpdomain.Limits, error) {
	config := mcpdomain.DefaultLimits().Config()
	for _, item := range []struct {
		key    string
		target *int
	}{
		{"MCP_REQUESTS_PER_MINUTE", &config.RequestsPerMinute},
		{"MCP_REQUESTS_PER_DAY", &config.RequestsPerDay},
		{"MCP_MAX_CONCURRENT", &config.MaxConcurrent},
	} {
		value := getenv(item.key)
		if value == "" {
			continue
		}

		number, err := strconv.ParseUint(value, 10, 31)
		if err != nil {
			return mcpdomain.Limits{}, fmt.Errorf("%s は正の整数で指定する: %w", item.key, err)
		}

		*item.target = int(number)
	}

	return mcpdomain.NewLimits(config)
}
