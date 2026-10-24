package main

import (
	"fmt"
	"strconv"

	"github.com/shigaresso/player-lens/internal/reviewmcp"
)

func readMCPLimits(getenv func(string) string) (reviewmcp.Limits, error) {
	config := reviewmcp.DefaultLimits().Config()
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
			return reviewmcp.Limits{}, fmt.Errorf("%s は正の整数で指定する: %w", item.key, err)
		}
		*item.target = int(number)
	}
	return reviewmcp.NewLimits(config)
}
