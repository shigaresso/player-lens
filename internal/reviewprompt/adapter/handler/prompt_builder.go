package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/handler/protocol"
	"github.com/shigaresso/player-lens/internal/reviewprompt/adapter/handler/logging"
	promptpage "github.com/shigaresso/player-lens/internal/reviewprompt/adapter/handler/page"
)

// PromptPath は画面の登録とログが共有する固定パスを表す。
const PromptPath = "/"

type promptBuilderHandler struct {
	logger *slog.Logger
	now    func() time.Time
	page   []byte
	policy string
}

// NewPromptBuilderHandler は固定の入力定義と埋め込み資産から並行利用可能な画面を生成する。
// 生成・リクエスト処理で Steam や MCP を呼ばず、入力・生成結果を保存しない。
// logger と now は有効な非 nil・並行利用可能な依存を渡す。
// 生成時にはログ・時計を実行せず、呼び出し側が GET/HEAD の厳密な PromptPath へ登録する。
func NewPromptBuilderHandler(logger *slog.Logger, now func() time.Time) (http.Handler, error) {
	page, policy, err := promptpage.Render(protocol.InputSchema())
	if err != nil {
		return nil, err
	}

	handler := promptBuilderHandler{page: page, policy: policy, logger: logger, now: now}
	return http.HandlerFunc(handler.Handle), nil
}

// Handle は画面の応答処理と所要時間を記録する。ログ失敗で応答を変更しない。
// URL 解析・生成・コピーはブラウザ内で実行し、panic は回復せず伝播する。
func (h promptBuilderHandler) Handle(w http.ResponseWriter, r *http.Request) {
	started := h.now()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", h.policy)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	response := promptpage.Serve(w, r, h.page)
	logging.PromptPage{Route: PromptPath, Method: r.Method, StatusCode: response.StatusCode, Duration: h.now().Sub(started), WriteErr: response.WriteErr}.Write(r.Context(), h.logger)
}
