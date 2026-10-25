package domain_test

import (
	"errors"
	"strings"
	"testing"

	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

func TestNewAccess(t *testing.T) {
	for _, tc := range []struct {
		name, value  string
		valid, local bool
	}{
		{"未指定ならローカル専用", "", true, true},
		{"HTTPSの公開URLを受け付ける", "https://reviews.example/tools", true, false},
		{"明示ポートを受け付ける", "https://reviews.example:8443/tools", true, false},
		{"IPv6を受け付ける", "https://[2001:db8::1]/tools", true, false},
		{"HTTPを拒否する", "http://reviews.example/tools", false, false},
		{"別パスを拒否する", "https://reviews.example/mcp", false, false},
		{"末尾スラッシュを拒否する", "https://reviews.example/tools/", false, false},
		{"encodedパスを拒否する", "https://reviews.example/%74ools", false, false},
		{"資格情報を拒否する", "https://secret:password@reviews.example/tools", false, false},
		{"queryを拒否する", "https://reviews.example/tools?secret=1", false, false},
		{"空queryを拒否する", "https://reviews.example/tools?", false, false},
		{"fragmentを拒否する", "https://reviews.example/tools#secret", false, false},
		{"空fragmentを拒否する", "https://reviews.example/tools#", false, false},
		{"空Hostを拒否する", "https:///tools", false, false},
		{"空白を拒否する", " https://reviews.example/tools", false, false},
		{"改行を拒否する", "https://reviews.example/tools\n", false, false},
		{"不正なURLを拒否する", "%", false, false},
		{"空ポートを拒否する", "https://reviews.example:/tools", false, false},
		{"ゼロポートを拒否する", "https://reviews.example:0/tools", false, false},
		{"範囲外ポートを拒否する", "https://reviews.example:65536/tools", false, false},
		{"英字ポートを拒否する", "https://reviews.example:bad/tools", false, false},
		{"空labelを拒否する", "https://reviews..example/tools", false, false},
		{"不正labelを拒否する", "https://-reviews.example/tools", false, false},
		{"末尾ハイフンを拒否する", "https://reviews-.example/tools", false, false},
		{"ワイルドカードを拒否する", "https://*.example/tools", false, false},
		{"長すぎるlabelを拒否する", "https://" + strings.Repeat("a", 64) + ".example/tools", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			access, err := mcpaccess.New(tc.value)
			if tc.valid {
				if err != nil || (access == nil) != tc.local {
					t.Fatalf("access=%v err=%v", access, err)
				}

				return
			}

			if access != nil || !errors.Is(err, mcpaccess.ErrInvalidConfig) {
				t.Fatalf("access=%v err=%v", access, err)
			}

			if strings.Contains(err.Error(), tc.value) {
				t.Fatal("設定値を診断へ出した")
			}
		})
	}
}
