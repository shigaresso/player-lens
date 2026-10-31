"""PR 規約ワークフローで追加の全検査が必要な編集を判定する。"""

import json
import os
from pathlib import Path
import sys


def needs_full_checks(event):
    """needs_full_checks は統合先などの編集や不明な編集情報で True を返す。"""
    action = event.get("action")
    if action == "edited":
        changes = event.get("changes")
        return not (isinstance(changes, dict) and changes and
                    set(changes).issubset({"title", "body"}))
    if action in ("opened", "synchronize", "reopened", "labeled", "unlabeled"):
        # コード変更の全検査は独立した conventions.yml が担当する。
        return False
    raise ValueError("未対応の PR イベント")


def main():
    try:
        event = json.loads(Path(sys.argv[1]).read_text(encoding="utf-8"))
        if not isinstance(event, dict):
            raise ValueError("PR イベントは JSON オブジェクトにする")
        value = "true" if needs_full_checks(event) else "false"
        with Path(os.environ["GITHUB_OUTPUT"]).open("a", encoding="utf-8") as output:
            output.write(f"full_checks={value}\n")
        return 0
    except (OSError, ValueError, KeyError, IndexError) as error:
        print(f"PR 検査範囲の判定が失敗した: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
