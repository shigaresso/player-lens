//go:build integration

package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	serveradapter "github.com/shigaresso/player-lens/cmd/server/adapter"
	serverhandler "github.com/shigaresso/player-lens/cmd/server/adapter/handler"
)

type trackedListener struct {
	net.Listener
	closed chan struct{}
	once   sync.Once
}

func TestShutdownDeadlineClosesConnection(t *testing.T) {
	started := make(chan struct{})
	finished := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(finished)
	})
	url, client, cancel, done, _ := startTestServer(t, handler, 0)
	requestDone := make(chan error, 1)
	go func() {
		response, err := client.Get(url)
		if response != nil {
			if closeErr := response.Body.Close(); closeErr != nil {
				t.Errorf("応答の解放に失敗した: %v", closeErr)
			}
		}

		requestDone <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("リクエストが開始しなかった")
	}

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("期限超過を報告しなかった: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("期限超過後も終了しなかった")
	}

	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("接続の context をキャンセルしなかった")
	}

	if err := <-requestDone; err == nil {
		t.Fatal("強制終了した接続を正常応答として扱った")
	}
}

func TestRunBindFailure(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Errorf("listener の解放に失敗した: %v", err)
		}
	})
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	err = run(t.Context(), port, slog.New(slog.NewJSONHandler(io.Discard, nil)))
	var operation *net.OpError
	if !errors.As(err, &operation) || operation.Op != "listen" {
		t.Fatalf("待ち受け失敗の原因が保持されなかった: %v", err)
	}
}

func (l *trackedListener) Close() error {
	err := l.Listener.Close()
	l.once.Do(func() { close(l.closed) })
	return err
}

func startTestServer(t *testing.T, handler http.Handler, timeout time.Duration) (
	string, *http.Client, context.CancelFunc, <-chan error, <-chan struct{},
) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	tracked := &trackedListener{Listener: listener, closed: make(chan struct{})}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- serveradapter.Serve(ctx, tracked, handler, timeout)
		close(done)
	}()
	transport := &http.Transport{}
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second}
	t.Cleanup(func() {
		cancel()
		transport.CloseIdleConnections()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("サーバーの goroutine が終了しなかった")
		}
	})
	return "http://" + listener.Addr().String(), client, cancel, done, tracked.closed
}

func TestServerHealthAndShutdown(t *testing.T) {
	healthHandler := serverhandler.NewHealthHandler(slog.New(slog.NewJSONHandler(io.Discard, nil)), time.Now)
	router := http.NewServeMux()
	serveradapter.RegisterRoutes(router, healthHandler, nil, nil)
	url, client, cancel, done, _ := startTestServer(t, router, time.Second)
	response, err := client.Get(url + "/healthz")
	if err != nil {
		t.Fatal(err)
	}

	body, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil || response.StatusCode != http.StatusOK || string(body) != "ok\n" {
		t.Fatalf("HTTP 応答: status=%d body=%q read=%v close=%v", response.StatusCode, body, readErr, closeErr)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("終了しなかった")
	}

	if response, err := client.Get(url + "/healthz"); err == nil {
		if closeErr := response.Body.Close(); closeErr != nil {
			t.Errorf("応答の解放に失敗した: %v", closeErr)
		}

		t.Fatal("終了後に新しい接続を受け付けた")
	}
}

func TestShutdownWaitsForActiveRequest(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	finish := sync.OnceFunc(func() { close(release) })
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-release:
			if _, err := w.Write([]byte("completed\n")); err != nil {
				t.Errorf("応答の書き込みに失敗した: %v", err)
			}
		case <-r.Context().Done():
		}
	})
	url, client, cancel, done, closed := startTestServer(t, handler, 3*time.Second)
	t.Cleanup(finish)
	result := make(chan error, 1)
	go func() {
		response, err := client.Get(url)
		if err == nil {
			body, readErr := io.ReadAll(response.Body)
			err = errors.Join(readErr, response.Body.Close())
			if string(body) != "completed\n" {
				err = errors.Join(err, io.ErrUnexpectedEOF)
			}
		}

		result <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("リクエストが開始しなかった")
	}

	cancel()
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("新規受付を停止しなかった")
	}

	select {
	case err := <-done:
		t.Fatalf("処理中のリクエストを待たず終了した: %v", err)
	default:
	}

	finish()
	if err := <-result; err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("リクエスト完了後も終了しなかった")
	}
}
