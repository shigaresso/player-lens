package main

import (
	"log"
	"net/http"
)

func newHandler() http.Handler {
	mux := http.NewServeMux()
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
