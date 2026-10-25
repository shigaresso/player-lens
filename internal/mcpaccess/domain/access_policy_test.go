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

	for _, tc := range []struct {
		name                 string
		access               *mcpaccess.Access
		host                 string
		initialized, allowed bool
	}{
		{"nil は未初期化として拒否する", nil, "reviews.example", false, false},
		{"ゼロ値は未初期化として拒否する", &mcpaccess.Access{}, "reviews.example", false, false},
		{"構築済み設定の公開 Host を正規化して許可する", configured, "REVIEWS.EXAMPLE:443", true, true},
		{"構築済み設定はローカル Host を許可する", configured, "127.0.0.1:8080", true, true},
		{"設定していない外部 Host を拒否する", configured, "other.example", true, false},
		{"不正な Host を拒否する", configured, "reviews.example/path", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.access.IsInitialized() != tc.initialized || tc.access.AllowsHost(tc.host) != tc.allowed {
				t.Fatalf("初期化済み=%v 許可=%v", tc.access.IsInitialized(), tc.access.AllowsHost(tc.host))
			}
		})
	}
}
