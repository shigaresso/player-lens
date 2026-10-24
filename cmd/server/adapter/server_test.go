package adapter

import (
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

type failedListener struct {
	cause  error
	closed atomic.Bool
}

func (l *failedListener) Accept() (net.Conn, error) { return nil, l.cause }
func (l *failedListener) Addr() net.Addr            { return &net.TCPAddr{} }
func (l *failedListener) Close() error              { l.closed.Store(true); return nil }

func TestServeFailure(t *testing.T) {
	cause := errors.New("待ち受け資源の故障")
	listener := &failedListener{cause: cause}
	err := Serve(t.Context(), listener, NewHandler(nil), time.Second)
	if !errors.Is(err, cause) {
		t.Fatalf("原因が保持されなかった: %v", err)
	}
	if !listener.closed.Load() {
		t.Fatal("listener が解放されなかった")
	}
}
