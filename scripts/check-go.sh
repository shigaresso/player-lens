#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mode="${1:-unit}"
case "$mode" in
  unit) ;;
  *) echo '検証対象は unit を指定する' >&2; exit 1 ;;
esac
docker compose run --rm --build -T verify bash scripts/go-check.sh "$mode"
