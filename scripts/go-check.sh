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
  *) echo '検証対象は unit を指定する' >&2; exit 1 ;;
esac
