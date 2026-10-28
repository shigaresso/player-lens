#!/usr/bin/env bash
set -euo pipefail
python3 scripts/check_secrets.py staged
bash scripts/check.sh
exec python3 scripts/staged_format.py
