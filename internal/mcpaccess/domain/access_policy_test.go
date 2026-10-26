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
		name            string
		access          mcpaccess.Access
		host            string
		public, allowed bool
	}{
		{"構築済み設定の公開 Host を正規化して許可する", configured, "REVIEWS.EXAMPLE:443", true, true},
		{"構築済み設定はローカル Host を許可する", configured, "127.0.0.1:8080", true, true},
		{"設定していない外部 Host を拒否する", configured, "other.example", true, false},
		{"不正な Host を拒否する", configured, "reviews.example/path", true, false},
		{"ローカル設定はlocalhostを許可する", local, "localhost", false, true},
		{"ローカル設定は大文字とポート付きlocalhostを許可する", local, "LOCALHOST:8080", false, true},
		{"ローカル設定はIPv4ループバックを許可する", local, "127.0.0.2:8080", false, true},
		{"ローカル設定はIPv6ループバックを許可する", local, "[0:0:0:0:0:0:0:1]:8080", false, true},
		{"ローカル設定は外部Hostを拒否する", local, "reviews.example", false, false},
		{"ローカル設定は空Hostを拒否する", local, "", false, false},
		{"ローカルの既存判定はポート文字列を正規化しない", local, "localhost:bad", false, true},
		{"公開設定はローカルHostでも不正ポートを拒否する", configured, "localhost:bad", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.access.IsPublic() != tc.public || tc.access.AllowsHost(tc.host) != tc.allowed {
				t.Fatalf("公開用=%v 許可=%v", tc.access.IsPublic(), tc.access.AllowsHost(tc.host))
			}
		})
	}
}

func TestAccessHostRejectionMessage(t *testing.T) {
	for _, tc := range []struct {
		name, publicURL, message string
	}{
		{"ローカル用の固定の拒否理由を返す", "", "ローカルの Host のみ利用できます"},
		{"公開用の固定の拒否理由を返す", "https://reviews.example/tools", "許可されていない Host です"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			access, err := mcpaccess.New(tc.publicURL)
			if err != nil {
				t.Fatal(err)
			}

			if got := access.HostRejectionMessage(); got != tc.message {
				t.Fatalf("拒否理由=%q; want=%q", got, tc.message)
			}
		})
	}
}
