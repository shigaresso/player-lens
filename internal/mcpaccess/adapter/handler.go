package adapter

import (
	"net/http"

	mcpaccess "github.com/shigaresso/player-lens/internal/mcpaccess/domain"
)

// Protect は生成済み Access の許可 Host と cross-origin を MCP の手前で検査する。
// Authorization は要求しない。next は /tools を担当し、healthz は外側に置く。
// deniedMessage は生成時に選んだ固定の公開用拒否文言を渡す。
// Access と next は生成成功した非 nil の値を渡す。存在の再検査は行わない。
func Protect(a mcpaccess.Access, deniedMessage string, next http.Handler) http.Handler {
	protected := http.NewCrossOriginProtection().Handler(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.AllowsHost(r.Host) {
			http.Error(w, deniedMessage, http.StatusForbidden)
			return
		}

		protected.ServeHTTP(w, r)
	})
}
