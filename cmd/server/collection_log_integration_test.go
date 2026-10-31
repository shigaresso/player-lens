//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type processLogWriter struct{ records chan []byte }

func (w processLogWriter) Write(data []byte) (int, error) {
	w.records <- bytes.Clone(data)
	return len(data), nil
}

func TestMCPCollectionLogOnStderr(t *testing.T) {
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
	command.Env = append(os.Environ(), "PORT="+strconv.Itoa(port), "MCP_PUBLIC_URL=", "MCP_REQUESTS_PER_MINUTE=6", "MCP_REQUESTS_PER_DAY=100", "MCP_MAX_CONCURRENT=2")
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

			return record
		case <-ctx.Done():
			t.Fatal("子プロセスのログを取得できなかった", ctx.Err())
			return nil
		}
	}
	if record := readRecord(); record["event"] != "server.started" {
		t.Fatalf("起動しなかった: %+v", record)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "collection-log-test", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: "http://127.0.0.1:" + strconv.Itoa(port) + "/tools"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := session.Close(); err != nil {
			t.Error(err)
		}
	})

	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "collect_reviews", Arguments: map[string]any{"app_id": 0, "private_field": "private input"}})
	if err != nil || result == nil || !result.IsError {
		t.Fatalf("不正入力の応答が異なる: result=%+v err=%v", result, err)
	}

	record := readRecord()
	if record["event"] != "review_collection.completed" || record["service.name"] != "player-lens" || record["level"] != "INFO" || record["error_code"] != "invalid_input" {
		t.Fatalf("起動側の logger を使わなかった: %+v", record)
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(data), "private") {
		t.Fatal("入力情報を stderr へ転載した")
	}

	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}

	runErr := <-done
	waited = true
	if runErr != nil {
		t.Fatalf("正常終了しなかった: %v", runErr)
	}

	if stdout.Len() != 0 || len(records) != 0 {
		t.Fatal("stdout へのログまたは余分なイベントがあった")
	}
}
