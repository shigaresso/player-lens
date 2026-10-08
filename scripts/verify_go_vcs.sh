#!/usr/bin/env bash
# Git 情報を利用できない環境で、共通設定と子 Go コマンドの動作を確認する。
set -euo pipefail
fixture="$(mktemp -d)"
trap 'rm -rf "$fixture"' EXIT
go_binary="$(command -v go)"
go_version="$(go env GOVERSION)"

for scenario in failed-git missing-git missing-checkout; do
  directory="$fixture/$scenario"
  mkdir -p "$directory/bin" "$directory/source"
  ln -s "$go_binary" "$directory/bin/go"
  if [[ "$scenario" != missing-git ]]; then
    cat > "$directory/bin/git" <<'SH'
#!/bin/sh
echo 'Git 情報の取得失敗を再現した' >&2
exit 128
SH
    chmod +x "$directory/bin/git"
  fi
  if [[ "$scenario" != missing-checkout ]]; then
    mkdir -p "$directory/source/.git/objects" "$directory/source/.git/refs"
    printf 'ref: refs/heads/main\n' > "$directory/source/.git/HEAD"
  fi
  printf 'module example.invalid/vcsfixture\n\ngo %s\n' "${go_version#go}" > "$directory/source/go.mod"
  cat > "$directory/source/main.go" <<'GO'
package main

func main() {}
GO
  cat > "$directory/source/main_test.go" <<'GO'
package main

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestChildBuild(t *testing.T) {
	command := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(t.TempDir(), "child"), ".")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("子 Go コマンドのビルド: %s err=%v", output, err)
	}
}
GO
  (
    cd "$directory/source"
    # PATH に Git がなくても Go の標準ライブラリだけで実行できる。
    PATH="$directory/bin" go list ./...
    PATH="$directory/bin" go build -o "$directory/program" .
    PATH="$directory/bin" go test -count=1 -timeout=60s ./...
  )
  echo "Git 非依存の Go 検査: $scenario が成功した"
done
