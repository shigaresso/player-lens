#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

case "$(uname -s)/$(uname -m)" in
  Darwin/arm64)
    actionlint_archive=actionlint_1.7.12_darwin_arm64.tar.gz
    actionlint_sha=aba9ced2dee8d27fecca3dc7feb1a7f9a52caefa1eb46f3271ea66b6e0e6953f
    lefthook_binary=lefthook_2.1.9_MacOS_arm64
    lefthook_sha=fd506e05954af2062ce320d59ac1f5bf13fad8d694694a72bc6ef91e8c284e3d
    ;;
  Linux/x86_64)
    actionlint_archive=actionlint_1.7.12_linux_amd64.tar.gz
    actionlint_sha=8aca8db96f1b94770f1b0d72b6dddcb1ebb8123cb3712530b08cc387b349a3d8
    lefthook_binary=lefthook_2.1.9_Linux_x86_64
    lefthook_sha=0d60b0d350c923963729574f6431171f0277788884ad0c6284fa0160c36e3877
    ;;
  *) echo '対応環境は macOS arm64 と Linux x86_64。別環境は導入 Issue で追加する。' >&2; exit 1 ;;
esac

task_temp=$(mktemp -d)
trap 'rm -rf "$task_temp"' EXIT
curl -fsSL --connect-timeout 10 --max-time 60 \
  "https://github.com/rhysd/actionlint/releases/download/v1.7.12/$actionlint_archive" -o "$task_temp/$actionlint_archive"
curl -fsSL --connect-timeout 10 --max-time 60 \
  "https://github.com/evilmartians/lefthook/releases/download/v2.1.9/$lefthook_binary" -o "$task_temp/$lefthook_binary"

python3 - "$task_temp/$actionlint_archive" "$actionlint_sha" "$task_temp/$lefthook_binary" "$lefthook_sha" <<'PY'
import hashlib
import sys
from pathlib import Path
for filename, expected in zip(sys.argv[1::2], sys.argv[2::2]):
    if hashlib.sha256(Path(filename).read_bytes()).hexdigest() != expected:
        raise SystemExit(f'checksum mismatch: {Path(filename).name}')
PY

tar -xzf "$task_temp/$actionlint_archive" -C "$task_temp" actionlint
mkdir -p .tools
install -m 755 "$task_temp/actionlint" .tools/actionlint
install -m 755 "$task_temp/$lefthook_binary" .tools/lefthook
echo '固定版・checksum を確認して .tools に配置した'
