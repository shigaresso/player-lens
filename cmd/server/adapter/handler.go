package adapter

import (
	"log"
	"net/http"
)

// NewHandler は healthz と指定した MCP の /tools をルーティングする。
// mcpHandler が nil なら /tools を登録しない。サーバーの起動と終了は呼び出し側が担当する。
func NewHandler(mcpHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	if mcpHandler != nil {
		mux.Handle("/tools", mcpHandler)
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if r.Method == http.MethodGet {
			if _, err := w.Write([]byte("ok\n")); err != nil {
				log.Printf("ヘルスチェック応答の書き込みに失敗した: %v", err)
			}
		}
	})
	return mux
}
