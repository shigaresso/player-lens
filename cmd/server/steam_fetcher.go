package main

import (
	"net/http"
	"time"

	"github.com/shigaresso/player-lens/internal/steam"
)

// newSteamFetcher はサーバー用の取得クライアントと idle 接続の終了関数を返す。
// 構築中は通信しない。成功後の終了関数は呼び出し側が実行し、構築失敗時はここで解放する。
func newSteamFetcher() (*steam.Client, func(), error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	fetcher, err := steam.NewClient(&http.Client{Transport: transport, Timeout: 10 * time.Second}, "")
	if err != nil {
		transport.CloseIdleConnections()
		return nil, nil, err
	}
	return fetcher, transport.CloseIdleConnections, nil
}
