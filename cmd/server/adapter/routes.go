package adapter

import (
	"net/http"

	serverhandler "github.com/shigaresso/player-lens/cmd/server/adapter/handler"
)

// RegisterRoutes は生成済みの handler を mux の /healthz・/tools・厳密な / に登録する。
// mux と healthHandler は有効な非 nil の値を渡す。mcpHandler が nil なら /tools は登録しない。
// 登録は受付開始前に行い、競合するパターンは事前に登録しない。
// 重複登録などの不正な登録は標準の http.ServeMux と同じく panic する。
// mux と handler の生成・実行、サーバーの起動と終了は呼び出し側が担当する。
// promptHandler が nil ならトップページを登録しない。
func RegisterRoutes(mux *http.ServeMux, healthHandler, mcpHandler, promptHandler http.Handler) {
	if mcpHandler != nil {
		mux.Handle("/tools", mcpHandler)
	}

	mux.Handle("GET "+serverhandler.HealthPath, healthHandler)
	if promptHandler != nil {
		mux.Handle("GET /{$}", promptHandler)
	}
}
