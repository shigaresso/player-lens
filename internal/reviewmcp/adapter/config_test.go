package adapter

import (
	"errors"
	"reflect"
	"testing"

	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

func TestNewHTTPConfig(t *testing.T) {
	for _, tc := range []struct {
		name, publicURL, host, deniedMessage       string
		valid, disableLocalhostProtection, allowed bool
	}{
		{"未指定ならローカル条件と文言とSDK保護を組み立てる", "", "reviews.example", "ローカルの Host のみ利用できます", true, false, false},
		{"公開URLなら対応する条件と文言とSDK設定を組み立てる", "https://reviews.example/tools", "REVIEWS.EXAMPLE:443", "許可されていない Host です", true, true, true},
		{"公開URLの非既定ポートを条件に保持する", "https://reviews.example:8443/tools", "reviews.example:8443", "許可されていない Host です", true, true, true},
		{"不正なURLでは部分的な設定を返さない", "http://reviews.example/tools", "", "", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config, err := NewHTTPConfig(tc.publicURL)
			if !tc.valid {
				if !errors.Is(err, mcpaccess.ErrInvalidConfig) || !reflect.ValueOf(config).IsZero() {
					t.Fatalf("config=%+v err=%v", config, err)
				}

				return
			}

			if err != nil || config.access == nil {
				t.Fatalf("config=%+v err=%v", config, err)
			}

			if config.access.AllowsHost(tc.host) != tc.allowed || !config.access.AllowsHost("localhost:8080") {
				t.Fatal("生成した許可条件が異なる")
			}

			if config.deniedMessage != tc.deniedMessage || config.sdkOptions.DisableLocalhostProtection != tc.disableLocalhostProtection {
				t.Fatalf("文言=%q SDK保護無効=%v", config.deniedMessage, config.sdkOptions.DisableLocalhostProtection)
			}

			if !config.sdkOptions.Stateless || !config.sdkOptions.JSONResponse || !config.sdkOptions.PropagateRequestCancellation || config.sdkOptions.MaxRequestBodyBytes != 16*1024 {
				t.Fatal("共通のSDK設定が異なる")
			}
		})
	}
}

func localHTTPConfig(t *testing.T) HTTPConfig {
	t.Helper()
	config, err := NewHTTPConfig("")
	if err != nil {
		t.Fatal(err)
	}

	return config
}
