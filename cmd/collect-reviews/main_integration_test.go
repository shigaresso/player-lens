//go:build integration

package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCLIProcessHelpAndInvalidInput(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "collect-reviews")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("CLIビルド: %s err=%v", output, err)
	}
	for _, test := range []struct {
		name string
		args []string
		code int
	}{
		{"helpはプロセスも正常終了する", []string{"--help"}, 0},
		{"不正入力はプロセスから2を返す", []string{"--app-id=0"}, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			command := exec.CommandContext(t.Context(), binary, test.args...)
			command.Stdout = &stdout
			command.Stderr = &stderr
			err := command.Run()
			code := 0
			if err != nil {
				var exit *exec.ExitError
				if !errors.As(err, &exit) {
					t.Fatal(err)
				}
				code = exit.ExitCode()
			}
			if code != test.code || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
	t.Run("診断先の閉じたpipeも終了コード1で報告する", func(t *testing.T) {
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		if err := reader.Close(); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := writer.Close(); err != nil {
				t.Error(err)
			}
		})
		command := exec.CommandContext(t.Context(), binary, "--help")
		command.Stderr = writer
		err = command.Run()
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			t.Fatalf("出力失敗の終了コード: %v", err)
		}
	})
}
