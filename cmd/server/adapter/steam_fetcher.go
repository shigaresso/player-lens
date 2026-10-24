package adapter

import (
	"net/http"
	"time"

	steamadapter "github.com/shigaresso/player-lens/internal/steam/adapter"
)

// NewSteamFetcher はサーバー用の取得クライアントと idle 接続の終了関数を返す。
// 構築中は通信しない。成功後の終了関数は呼び出し側が実行し、構築失敗時はここで解放する。
func NewSteamFetcher() (*steamadapter.Client, func(), error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	fetcher, err := steamadapter.NewClient(&http.Client{Transport: transport, Timeout: 10 * time.Second}, "")
	if err != nil {
		transport.CloseIdleConnections()
		return nil, nil, err
	}
	return fetcher, transport.CloseIdleConnections, nil
}
