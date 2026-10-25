package main

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestRunRejectsInvalidPort(t *testing.T) {
	if err := run(context.Background(), "invalid", slog.New(slog.NewJSONHandler(io.Discard, nil))); err == nil {
		t.Fatal("不正な PORT を受け付けた")
	}
}
