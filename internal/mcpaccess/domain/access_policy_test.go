package domain_test

import (
	"testing"

	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

func TestAccessPolicy(t *testing.T) {
	configured, err := mcpaccess.New("https://reviews.example/tools")
	if err != nil {
		t.Fatal(err)
	}

	local, err := mcpaccess.New("")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name                         string
		access                       *mcpaccess.Access
		host                         string
		initialized, public, allowed bool
	}{
		{"nil は未初期化として拒否する", nil, "reviews.example", false, false, false},
		{"ゼロ値は未初期化として拒否する", &mcpaccess.Access{}, "reviews.example", false, false, false},
		{"構築済み設定の公開 Host を正規化して許可する", configured, "REVIEWS.EXAMPLE:443", true, true, true},
		{"構築済み設定はローカル Host を許可する", configured, "127.0.0.1:8080", true, true, true},
		{"設定していない外部 Host を拒否する", configured, "other.example", true, true, false},
		{"不正な Host を拒否する", configured, "reviews.example/path", true, true, false},
		{"ローカル設定はlocalhostを許可する", local, "localhost", true, false, true},
		{"ローカル設定は大文字とポート付きlocalhostを許可する", local, "LOCALHOST:8080", true, false, true},
		{"ローカル設定はIPv4ループバックを許可する", local, "127.0.0.2:8080", true, false, true},
		{"ローカル設定はIPv6ループバックを許可する", local, "[0:0:0:0:0:0:0:1]:8080", true, false, true},
		{"ローカル設定は外部Hostを拒否する", local, "reviews.example", true, false, false},
		{"ローカル設定は空Hostを拒否する", local, "", true, false, false},
		{"ローカルの既存判定はポート文字列を正規化しない", local, "localhost:bad", true, false, true},
		{"公開設定はローカルHostでも不正ポートを拒否する", configured, "localhost:bad", true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.access.IsInitialized() != tc.initialized || tc.access.IsPublic() != tc.public || tc.access.AllowsHost(tc.host) != tc.allowed {
				t.Fatalf("初期化済み=%v 許可=%v", tc.access.IsInitialized(), tc.access.AllowsHost(tc.host))
			}
		})
	}
}
