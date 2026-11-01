package handler

import (
	"bytes"
	"net/http"
	"time"

	"github.com/shigaresso/player-lens/internal/reviewmcp/adapter/handler/protocol"
	promptpage "github.com/shigaresso/player-lens/internal/reviewprompt/adapter/handler/page"
)

type promptBuilderHandler struct {
	page   []byte
	policy string
}

// NewPromptBuilderHandler は固定の入力定義と埋め込み資産から並行利用可能な画面を生成する。
// 生成・リクエスト処理で Steam や MCP を呼ばず、入力・生成結果を保存しない。
// 呼び出し側が GET/HEAD の厳密なルートパスへ登録する。
func NewPromptBuilderHandler() (http.Handler, error) {
	page, policy, err := promptpage.Render(protocol.InputSchema())
	if err != nil {
		return nil, err
	}

	handler := promptBuilderHandler{page: page, policy: policy}
	return http.HandlerFunc(handler.Handle), nil
}

// Handle は画面を返す。URL 解析・生成・コピーはブラウザ内で実行する。
func (h promptBuilderHandler) Handle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", h.policy)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(h.page))
}
