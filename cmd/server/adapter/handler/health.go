package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/shigaresso/player-lens/cmd/server/adapter/handler/logging"
)

// HealthPath は生存確認の登録とログが共有する公開パスを表す。
const HealthPath = "/healthz"

type healthHandler struct {
	logger *slog.Logger
	now    func() time.Time
}

// NewHealthHandler は生存確認の HTTP 応答・計測・ログを担当する handler を生成する。
// logger と now は有効な非 nil かつ並行利用可能な依存を渡す。
// 生成時には時刻取得やログ出力を行わず、経路登録は呼び出し側が担当する。
// 呼び出し側は GET/HEAD の HealthPath に登録してメソッドとパスを制限する。
func NewHealthHandler(logger *slog.Logger, now func() time.Time) http.Handler {
	return healthHandler{logger: logger, now: now}
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

	logging.HealthCheck{Route: HealthPath, Method: r.Method, Duration: h.now().Sub(started), WriteErr: writeErr}.Write(r.Context(), h.logger)
}
