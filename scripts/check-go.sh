#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
mode="${1:-unit}"
case "$mode" in
  format|format-fix|unit|integration) ;;
  *) echo '対象は format、format-fix、unit、integration を指定する' >&2; exit 1 ;;
esac
mount_args=()
if [[ "$mode" == format-fix ]]; then
  mount_args=(--volume "$PWD:/src:rw")
fi
docker compose run --rm --no-deps --build -T "${mount_args[@]}" verify bash scripts/go-check.sh "$mode"
