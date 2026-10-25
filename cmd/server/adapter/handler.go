package adapter

import (
	"log/slog"
	"net/http"
	"time"
)

// NewHandler は healthz と指定した MCP の /tools をルーティングする。
// mcpHandler が nil なら /tools を登録しない。サーバーの起動と終了は呼び出し側が担当する。
// logger と now は呼び出し側が生成した有効な非 nil の依存を渡し、並行利用に対応させる。
// 生存確認の通常終了を request context 付きで記録し、ログ出力エラーは HTTP 応答へ反映しない。
func NewHandler(mcpHandler http.Handler, logger *slog.Logger, now func() time.Time) http.Handler {
	mux := http.NewServeMux()
	if mcpHandler != nil {
		mux.Handle("/tools", mcpHandler)
	}

	mux.Handle("GET /healthz", healthHandler{logger: logger, now: now})
	return mux
}
