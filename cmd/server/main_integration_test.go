//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStartupFailureStructuredLog(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	binary := filepath.Join(t.TempDir(), "server")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("serverのビルドに失敗した: %v: %s", err, output)
	}

	command := exec.CommandContext(ctx, binary)
	command.Env = append(os.Environ(), "PORT=invalid")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || exitError.ExitCode() != 1 || stdout.Len() != 0 {
		t.Fatalf("終了契約を変更した: err=%v stdout=%q", err, stdout.String())
	}

	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stderr.Bytes()), &record); err != nil {
		t.Fatalf("起動失敗が単一のJSONではない: %v: %s", err, stderr.String())
	}

	for key, want := range map[string]any{
		"level": "ERROR", "event": "server.failed", "service.name": "player-lens",
		"msg": "HTTP サーバーの実行に失敗した",
	} {
		if record[key] != want {
			t.Errorf("%s=%v; want=%v", key, record[key], want)
		}
	}

	cause, ok := record["error"].(string)
	if !ok || !strings.Contains(cause, "PORT") {
		t.Fatalf("起動の診断を失った: %v", record["error"])
	}
}
