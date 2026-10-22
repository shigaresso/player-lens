package mcpaccess

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// Protect は許可 Host と cross-origin を MCP の手前で検査する。
// Authorization は要求しない。next は /tools を担当し、healthz は外側に置く。
// ゼロ値・nil の設定やハンドラーは ErrInvalidConfig とする。
func (a *Access) Protect(next http.Handler) (http.Handler, error) {
	if a == nil || a.authority == "" || next == nil {
		return nil, fmt.Errorf("公開 MCP の保護には有効な設定とハンドラーが必要: %w", ErrInvalidConfig)
	}
	protected := http.NewCrossOriginProtection().Handler(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authority, err := canonicalAuthority(r.Host)
		if err != nil || !a.allowedAuthority(authority) {
			http.Error(w, "許可されていない Host です", http.StatusForbidden)
			return
		}
		protected.ServeHTTP(w, r)
	}), nil
}

func (a *Access) allowedAuthority(authority string) bool {
	if authority == a.authority {
		return true
	}
	name, _, _ := strings.Cut(authority, "|")
	if name == "localhost" {
		return true
	}
	ip := net.ParseIP(name)
	return ip != nil && ip.IsLoopback()
}
