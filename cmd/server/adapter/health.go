package adapter

import (
	"log/slog"
	"net/http"
	"time"
)

type healthHandler struct {
	logger *slog.Logger
	now    func() time.Time
}

// ServeHTTP は生存確認の応答と所要時間を記録する。ログの失敗で応答を変更しない。
// panic は回復せず、request context をそのままログへ渡す。
func (h healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	started := h.now()
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	var writeErr error
	if r.Method == http.MethodGet {
		_, writeErr = w.Write([]byte("ok\n"))
	}

	healthCheckLog{method: r.Method, duration: h.now().Sub(started), writeErr: writeErr}.write(r.Context(), h.logger)
}
