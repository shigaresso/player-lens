package main

import (
	"context"
	"testing"
)

func TestRunRejectsInvalidPort(t *testing.T) {
	if err := run(context.Background(), "invalid"); err == nil {
		t.Fatal("不正な PORT を受け付けた")
	}
}
