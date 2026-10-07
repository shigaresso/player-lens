#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
bash scripts/check-go.sh unit
bash scripts/check-go.sh integration
python3 scripts/check_image.py
