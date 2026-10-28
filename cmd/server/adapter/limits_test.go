package adapter

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	mcpdomain "github.com/shigaresso/player-lens/internal/reviewmcp/domain"
	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestReadMCPLimits(t *testing.T) {
	for _, tc := range []struct {
		name   string
		values map[string]string
		want   mcpdomain.LimitsConfig
		valid  bool
	}{
		{"未指定なら既定値", nil, mcpdomain.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 2}, true},
		{"指定した上限を使う", map[string]string{"MCP_REQUESTS_PER_MINUTE": "3", "MCP_REQUESTS_PER_DAY": "20", "MCP_MAX_CONCURRENT": "1"}, mcpdomain.LimitsConfig{RequestsPerMinute: 3, RequestsPerDay: 20, MaxConcurrent: 1}, true},
		{"先頭のゼロを正規化する", map[string]string{"MCP_REQUESTS_PER_MINUTE": "006"}, mcpdomain.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 2}, true},
		{"日次だけ指定して他の既定値を維持する", map[string]string{"MCP_REQUESTS_PER_DAY": "20"}, mcpdomain.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 20, MaxConcurrent: 2}, true},
		{"同時数だけ指定して他の既定値を維持する", map[string]string{"MCP_MAX_CONCURRENT": "3"}, mcpdomain.LimitsConfig{RequestsPerMinute: 6, RequestsPerDay: 100, MaxConcurrent: 3}, true},
		{"各項目の下限を受け付ける", map[string]string{"MCP_REQUESTS_PER_MINUTE": "1", "MCP_REQUESTS_PER_DAY": "1", "MCP_MAX_CONCURRENT": "1"}, mcpdomain.LimitsConfig{RequestsPerMinute: 1, RequestsPerDay: 1, MaxConcurrent: 1}, true},
		{"各項目の31bit上限を受け付ける", map[string]string{"MCP_REQUESTS_PER_MINUTE": "2147483647", "MCP_REQUESTS_PER_DAY": "2147483647", "MCP_MAX_CONCURRENT": "2147483647"}, mcpdomain.LimitsConfig{RequestsPerMinute: 2147483647, RequestsPerDay: 2147483647, MaxConcurrent: 2147483647}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			limits, err := ReadMCPLimits(func(key string) string { return tc.values[key] })
			if err != nil || limits.Config() != tc.want {
				t.Fatalf("limits=%+v err=%v", limits, err)
			}
		})
	}

	for _, key := range []string{"MCP_REQUESTS_PER_MINUTE", "MCP_REQUESTS_PER_DAY", "MCP_MAX_CONCURRENT"} {
		for _, input := range []string{"0", "-1", "+1", " 1", "1.5", "none", "2147483648"} {
			t.Run(key+"の不正値"+input+"を拒否する", func(t *testing.T) {
				limits, err := ReadMCPLimits(func(name string) string {
					if name == key {
						return input
					}

					return ""
				})
				if err == nil || limits != (mcpdomain.Limits{}) {
					t.Fatalf("limits=%+v err=%v", limits, err)
				}
			})
		}
	}
}

func TestReadMCPLimitsReadsEnvironment(t *testing.T) {
	var keys []string
	_, err := ReadMCPLimits(func(key string) string {
		keys = append(keys, key)
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}

	if got := strings.Join(keys, ","); got != "MCP_REQUESTS_PER_MINUTE,MCP_REQUESTS_PER_DAY,MCP_MAX_CONCURRENT" {
		t.Fatalf("必要な3項目を順に1回ずつ読む: keys=%v", keys)
	}
}

func TestReadMCPLimitsFailureCause(t *testing.T) {
	tests := []struct {
		name    string
		values  map[string]string
		cause   error
		message string
	}{
		{"構文不正は環境変数名と変換原因を保持する", map[string]string{"MCP_REQUESTS_PER_MINUTE": "bad"}, strconv.ErrSyntax, "MCP_REQUESTS_PER_MINUTE"},
		{"31bit超過は環境変数名と範囲の原因を保持する", map[string]string{"MCP_MAX_CONCURRENT": "2147483648"}, strconv.ErrRange, "MCP_MAX_CONCURRENT"},
		{"ゼロはドメインの正数条件で拒否する", map[string]string{"MCP_REQUESTS_PER_DAY": "0"}, steamdomain.ErrInvalidArgument, "MCP の24時間の上限は正の整数で指定する"},
		{"複数の構文不正では先の項目の原因を返す", map[string]string{"MCP_REQUESTS_PER_MINUTE": "bad", "MCP_REQUESTS_PER_DAY": "bad"}, strconv.ErrSyntax, "MCP_REQUESTS_PER_MINUTE"},
		{"正数条件より全項目の数値変換を先に確認する", map[string]string{"MCP_REQUESTS_PER_MINUTE": "0", "MCP_MAX_CONCURRENT": "bad"}, strconv.ErrSyntax, "MCP_MAX_CONCURRENT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits, err := ReadMCPLimits(func(key string) string { return tt.values[key] })
			if !errors.Is(err, tt.cause) || !strings.Contains(err.Error(), tt.message) || limits != (mcpdomain.Limits{}) {
				t.Fatalf("ゼロ値と既存の診断・原因を返す: limits=%v err=%v; cause=%v message=%q", limits, err, tt.cause, tt.message)
			}
		})
	}
}
