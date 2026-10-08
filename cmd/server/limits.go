package main

import (
	"fmt"
	"strconv"

	"github.com/shigaresso/player-lens/internal/reviewmcp"
)

func readMCPLimits(getenv func(string) string) (reviewmcp.Limits, error) {
	limits := reviewmcp.DefaultLimits()
	for _, item := range []struct {
		key    string
		target *int
	}{
		{"MCP_REQUESTS_PER_MINUTE", &limits.RequestsPerMinute},
		{"MCP_REQUESTS_PER_DAY", &limits.RequestsPerDay},
		{"MCP_MAX_CONCURRENT", &limits.MaxConcurrent},
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
	if err := limits.Validate(); err != nil {
		return reviewmcp.Limits{}, err
	}
	return limits, nil
}
