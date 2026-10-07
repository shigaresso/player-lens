#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

python3 scripts/validate.py pre-push "$1"
bash scripts/check.sh
bash scripts/check-app.sh
