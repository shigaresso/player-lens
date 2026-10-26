package adapter

import (
	"fmt"
	"net/http"

	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

// Protect は許可 Host と cross-origin を MCP の手前で検査する。
// Authorization は要求しない。next は /tools を担当し、healthz は外側に置く。
// ゼロ値・nil の設定やハンドラーは ErrInvalidConfig とする。
func Protect(a *mcpaccess.Access, next http.Handler) (http.Handler, error) {
	if !a.IsInitialized() || next == nil {
		return nil, fmt.Errorf("公開 MCP の保護には有効な設定とハンドラーが必要: %w", mcpaccess.ErrInvalidConfig)
	}

	protected := http.NewCrossOriginProtection().Handler(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.AllowsHost(r.Host) {
			http.Error(w, "許可されていない Host です", http.StatusForbidden)
			return
		}

		protected.ServeHTTP(w, r)
	}), nil
}
