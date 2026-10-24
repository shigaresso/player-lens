//go:build integration

package adapter

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	steamdomain "github.com/shigaresso/player-lens/internal/steam/domain"
)

func TestNewSteamFetcherCleanup(t *testing.T) {
	idle := make(chan net.Conn, 2)
	closed := make(chan net.Conn, 2)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := io.WriteString(w, `{"response":{"reviews":[]}}`); err != nil {
			t.Errorf("テスト用のレビュー応答を書き込めなかった: %v", err)
		}
	}))
	server.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		switch state {
		case http.StateIdle:
			idle <- conn
		case http.StateClosed:
			closed <- conn
		}
	}
	server.StartTLS()
	t.Cleanup(server.Close)

	transport := server.Client().Transport.(*http.Transport).Clone()
	t.Cleanup(transport.CloseIdleConnections)
	transport.Proxy = nil
	transport.TLSClientConfig.ServerName = server.Listener.Addr().(*net.TCPAddr).IP.String()
	dialer := &net.Dialer{}
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, server.Listener.Addr().String())
	}

	// 既定 Transport を差し替えるため並列実行せず、終了時に元へ戻す。
	originalTransport := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = originalTransport })
	http.DefaultTransport = transport

	fetcher, closeIdleConnections, err := NewSteamFetcher()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(closeIdleConnections)
	fetchPage := func() {
		t.Helper()
		if _, err := fetcher.FetchReviewsPage(t.Context(), 440, steamdomain.PageOptions{}); err != nil {
			t.Fatalf("ローカル TLS サーバーから取得できなかった: %v", err)
		}
	}

	fetchPage()
	connection := waitForSteamConnection(t, idle, "取得後に接続が idle になる")
	fetchPage()
	if reused := waitForSteamConnection(t, idle, "再取得後に接続が idle になる"); reused != connection {
		t.Fatal("終了関数を呼ぶ前に同じ接続が再利用されなかった")
	}

	closeIdleConnections()
	if released := waitForSteamConnection(t, closed, "終了関数で接続が閉じる"); released != connection {
		t.Fatal("取得に使った接続が閉じられなかった")
	}
}

func waitForSteamConnection(t *testing.T, connections <-chan net.Conn, expected string) net.Conn {
	t.Helper()
	select {
	case connection := <-connections:
		return connection
	case <-time.After(5 * time.Second):
		t.Fatalf("%sことを確認できなかった", expected)
		return nil
	}
}
