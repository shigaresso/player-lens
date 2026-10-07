#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
case "${1:-unit}" in
  unit)
    unformatted="$(gofmt -l .)"
    if [[ -n "$unformatted" ]]; then
      echo 'Go の整形差分を修正する:' >&2
      echo "$unformatted" >&2
      exit 1
    fi
    go vet ./...
    go test -count=1 ./...
    ;;
  integration)
    test_files="$(go list -buildvcs=false -tags=integration -f '{{range .TestGoFiles}}{{println .}}{{end}}{{range .XTestGoFiles}}{{println .}}{{end}}' ./...)"
    if ! grep '_integration_test.go$' <<< "$test_files" > /dev/null; then
      echo '結合テストが存在しない' >&2
      exit 1
    fi
    go test -tags=integration -race -count=1 ./...
    ;;
  *) echo '検証対象は unit または integration を指定する' >&2; exit 1 ;;
esac
