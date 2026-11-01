//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestPromptPageLogOnStderr(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	binary := filepath.Join(t.TempDir(), "server")
	if output, err := exec.CommandContext(ctx, "go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("serverのビルドに失敗した: %v: %s", err, output)
	}

	reservation, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	port := reservation.Addr().(*net.TCPAddr).Port
	if err := reservation.Close(); err != nil {
		t.Fatal(err)
	}

	records := make(chan []byte, 16)
	var stdout bytes.Buffer
	command := exec.CommandContext(ctx, binary)
	command.Env = append(os.Environ(), "PORT="+strconv.Itoa(port), "MCP_PUBLIC_URL=")
	command.Stdout, command.Stderr = &stdout, processLogWriter{records: records}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}

	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	waited := false
	t.Cleanup(func() {
		if !waited {
			cancel()
			<-done
		}
	})
	readRecord := func() map[string]any {
		t.Helper()
		select {
		case data := <-records:
			var record map[string]any
			if err := json.Unmarshal(bytes.TrimSpace(data), &record); err != nil {
				t.Fatal(err)
			}

			if strings.Contains(string(data), "private") {
				t.Fatal("受信情報をログへ含めた")
			}

			return record
		case <-ctx.Done():
			t.Fatal("子プロセスのログを取得できなかった", ctx.Err())
			return nil
		}
	}
	if record := readRecord(); record["event"] != "server.started" {
		t.Fatal("起動イベントがない")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	for _, tc := range []struct {
		name, method, path, header, value string
		status                            int
		event                             string
	}{
		{"GET", "GET", "/?private-query=value", "", "", 200, "prompt_page.completed"},
		{"HEAD", "HEAD", "/", "", "", 200, "prompt_page.completed"},
		{"部分応答", "GET", "/", "Range", "bytes=0-3", 206, "prompt_page.completed"},
		{"未変更", "GET", "/", "If-None-Match", "*", 304, "prompt_page.completed"},
		{"範囲外", "GET", "/", "Range", "bytes=99999999-", 416, "prompt_page.completed"},
		{"POSTは記録しない", "POST", "/", "", "", 405, ""},
		{"未定義パスは記録しない", "GET", "/private-missing", "", "", 404, ""},
		{"生存確認のイベントと分ける", "GET", "/healthz", "", "", 200, "health_check.completed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request, err := http.NewRequestWithContext(ctx, tc.method, "http://127.0.0.1:"+strconv.Itoa(port)+tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}

			request.Header.Set("User-Agent", "private-client")
			if tc.header != "" {
				request.Header.Set(tc.header, tc.value)
			}

			response, err := client.Do(request)
			if err != nil {
				t.Fatal(err)
			}

			_, readErr := io.Copy(io.Discard, response.Body)
			closeErr := response.Body.Close()
			if response.StatusCode != tc.status || readErr != nil || closeErr != nil {
				t.Fatalf("status=%d read=%v close=%v", response.StatusCode, readErr, closeErr)
			}

			if tc.event != "" {
				record := readRecord()
				if record["event"] != tc.event || record["http.response.status_code"] != float64(tc.status) || record["service.name"] != "player-lens" {
					t.Fatalf("応答とイベントが一致しない: %+v", record)
				}
			}
		})
	}

	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	runErr := <-done
	waited = true
	if runErr != nil || stdout.Len() != 0 || len(records) != 0 {
		t.Fatalf("終了・出力先・イベント数が異なる: err=%v stdout=%d extra=%d", runErr, stdout.Len(), len(records))
	}
}
