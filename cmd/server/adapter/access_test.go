package adapter

import (
	"errors"
	"testing"

	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

func TestReadMCPAccess(t *testing.T) {
	for _, tc := range []struct {
		name, value  string
		valid, local bool
	}{
		{"未指定ならローカル専用", "", true, true},
		{"公開URLだけで匿名公開を設定できる", "https://reviews.example/tools", true, false},
		{"不正なURLを拒否する", "http://reviews.example/tools", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			access, err := ReadMCPAccess(func(name string) string {
				if name != "MCP_PUBLIC_URL" {
					t.Fatalf("不要な設定を要求した: %s", name)
				}

				return tc.value
			})
			if tc.valid {
				if err != nil || access == nil || !access.IsInitialized() || access.IsPublic() == tc.local {
					t.Fatalf("access=%v err=%v", access, err)
				}

				return
			}

			if access != nil || !errors.Is(err, mcpaccess.ErrInvalidConfig) {
				t.Fatalf("access=%v err=%v", access, err)
			}
		})
	}
}
