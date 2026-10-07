#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export PATH="$PWD/.tools:$PATH"

python3 - <<'PY'
import unittest
suite = unittest.defaultTestLoader.discover('scripts', pattern='test_*.py')
if not suite.countTestCases():
    raise SystemExit('規約検査のテストが存在しない')
raise SystemExit(not unittest.TextTestRunner(verbosity=2).run(suite).wasSuccessful())
PY
git diff --check
git diff --cached --check
lefthook validate
actionlint -shellcheck=''
bash -n scripts/check.sh scripts/pre-push.sh scripts/install-tools.sh .lefthook/pre-commit/check.sh .lefthook/pre-push/check.sh
