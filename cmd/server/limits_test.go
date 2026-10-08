package main

import (
	"testing"

	"github.com/shigaresso/player-lens/internal/reviewmcp"
)

func TestReadMCPLimits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values map[string]string
		want   reviewmcp.Limits
		valid  bool
	}{
		{"未指定なら既定値", nil, reviewmcp.Limits{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 2}, true},
		{"指定した上限を使う", map[string]string{"MCP_REQUESTS_PER_MINUTE": "3", "MCP_REQUESTS_PER_DAY": "20", "MCP_MAX_CONCURRENT": "1"}, reviewmcp.Limits{RequestsPerMinute: 3, RequestsPerDay: 20, MaxConcurrent: 1}, true},
		{"先頭のゼロを正規化する", map[string]string{"MCP_REQUESTS_PER_MINUTE": "006"}, reviewmcp.Limits{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 2}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits, err := readMCPLimits(func(key string) string { return tc.values[key] })
			if err != nil || limits != tc.want {
				t.Fatalf("limits=%+v err=%v", limits, err)
			}
		})
	}
	for _, key := range []string{"MCP_REQUESTS_PER_MINUTE", "MCP_REQUESTS_PER_DAY", "MCP_MAX_CONCURRENT"} {
		for _, input := range []string{"0", "-1", "+1", " 1", "1.5", "none", "2147483648"} {
			t.Run(key+"の不正値"+input+"を拒否する", func(t *testing.T) {
				limits, err := readMCPLimits(func(name string) string {
					if name == key {
						return input
					}
					return ""
				})
				if err == nil || limits != (reviewmcp.Limits{}) {
					t.Fatalf("limits=%+v err=%v", limits, err)
				}
			})
		}
	}
}
