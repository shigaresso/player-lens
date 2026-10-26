package adapter

import (
	"errors"
	"reflect"
	"testing"

	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

func TestReadMCPHTTPConfig(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		valid       bool
	}{
		{"未指定ならローカル専用", "", true},
		{"公開URLだけで匿名公開を設定できる", "https://reviews.example/tools", true},
		{"不正なURLを拒否する", "http://reviews.example/tools", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, err := ReadMCPHTTPConfig(func(name string) string {
				if name != "MCP_PUBLIC_URL" {
					t.Fatalf("不要な設定を要求した: %s", name)
				}

				return tc.value
			})
			if tc.valid {
				if err != nil || reflect.ValueOf(config).IsZero() {
					t.Fatalf("config=%+v err=%v", config, err)
				}

				return
			}

			if !reflect.ValueOf(config).IsZero() || !errors.Is(err, mcpaccess.ErrInvalidConfig) {
				t.Fatalf("config=%+v err=%v", config, err)
			}
		})
	}
}
