#!/usr/bin/env bash
set -euo pipefail
bash scripts/check.sh
exec python3 scripts/staged_format.py
