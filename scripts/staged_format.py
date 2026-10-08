"""コミット対象の Go ソースを index から取り出して整形を検査する。"""

import os
import subprocess
import sys
import tempfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def check_staged(repo, *, formatter):
    """check_staged は変更された Go の一時コピーだけを formatter に渡す。"""
    result = subprocess.run(["git", "diff", "--cached", "--name-only",
                             "--diff-filter=ACMR", "-z", "--", "*.go"],
                            cwd=repo, check=True, capture_output=True, timeout=30)
    paths = [os.fsdecode(path) for path in result.stdout.split(b"\0") if path]
    if not paths:
        return
    with tempfile.TemporaryDirectory(prefix="player-lens-format-") as directory:
        snapshot = Path(directory)
        for path in paths:
            source = subprocess.run(["git", "show", f":{path}"], cwd=repo,
                                    check=True, capture_output=True, timeout=30).stdout
            destination = snapshot / path
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(source)
        formatter(repo, snapshot)


def format_snapshot(repo, snapshot, *, run=subprocess.run):
    """format_snapshot は固定版 Go の gofmt で一時コピーを読み取り専用で検査する。"""
    run(["docker", "compose", "build", "verify"], cwd=repo, check=True,
        capture_output=True, text=True, timeout=600)
    result = run(["docker", "compose", "run", "--rm", "--no-deps", "-T",
                  "--volume", f"{snapshot}:/format-input:ro",
                  "verify", "gofmt", "-l", "/format-input"],
                 cwd=repo, check=True, capture_output=True, text=True, timeout=60)
    if result.stdout.strip():
        paths = result.stdout.replace("/format-input/", "").strip()
        raise ValueError(f"ステージ済み Go の整形差分を修正する:\n{paths}")


def main():
    try:
        check_staged(ROOT, formatter=format_snapshot)
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        print(f"Go のコミット前整形検査が失敗した: {error}", file=sys.stderr)
        if isinstance(error, subprocess.CalledProcessError):
            print((error.stderr or "")[-3000:], file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
