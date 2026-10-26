package domain_test

import (
	"testing"

	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

func TestAccessPolicy(t *testing.T) {
	configured, err := mcpaccess.NewPublic("https://reviews.example/tools")
	if err != nil {
		t.Fatal(err)
	}

	local := mcpaccess.NewLocal()

	for _, tc := range []struct {
		name    string
		access  mcpaccess.Access
		host    string
		allowed bool
	}{
		{"構築済み設定の公開 Host を正規化して許可する", configured, "REVIEWS.EXAMPLE:443", true},
		{"構築済み設定はローカル Host を許可する", configured, "127.0.0.1:8080", true},
		{"設定していない外部 Host を拒否する", configured, "other.example", false},
		{"不正な Host を拒否する", configured, "reviews.example/path", false},
		{"ローカル設定はlocalhostを許可する", local, "localhost", true},
		{"ローカル設定は大文字とポート付きlocalhostを許可する", local, "LOCALHOST:8080", true},
		{"ローカル設定はIPv4ループバックを許可する", local, "127.0.0.2:8080", true},
		{"ローカル設定はIPv6ループバックを許可する", local, "[0:0:0:0:0:0:0:1]:8080", true},
		{"ローカル設定は外部Hostを拒否する", local, "reviews.example", false},
		{"ローカル設定は空Hostを拒否する", local, "", false},
		{"ローカルの既存判定はポート文字列を正規化しない", local, "localhost:bad", true},
		{"公開設定はローカルHostでも不正ポートを拒否する", configured, "localhost:bad", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.access.AllowsHost(tc.host) != tc.allowed {
				t.Fatalf("許可=%v; want=%v", tc.access.AllowsHost(tc.host), tc.allowed)
			}
		})
	}
}
