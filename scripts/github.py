"""Issue・PR の作成手順。検査と GitHub への書き込みを同じ入口で扱う。"""

import argparse
import re
import subprocess
import sys
from pathlib import Path
from urllib.parse import quote

import conventions
from validate import github_api


def create(kind, repo, data, *, api, send, head=None, base="main", check_only=False):
    """create は記載内容を受け取り、作成した URL の出力を返す。"""
    data = {"title": data["title"], "body": data["body"], "labels": list(data["labels"])}
    errors = (conventions.validate_pr if kind == "pr" else conventions.validate_issue)(data)
    if errors:
        raise ValueError("; ".join(errors))
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repo):
        raise ValueError("リポジトリを OWNER/REPO の形式で指定する")
    labels = [label["name"] if isinstance(label, dict) else label
              for label in data["labels"]]
    if any("," in label for label in labels):
        raise ValueError("ラベル名にカンマを含めない。複数ラベルは --label を繰り返す")
    available = {label["name"] for label in api(f"repos/{repo}/labels", paginate=True)}
    if set(labels) - available:
        raise ValueError("指定したラベルが存在しない。作成前にラベルを整備する")
    if kind == "pr":
        base_sha = api(f"repos/{repo}/git/ref/heads/{quote(base, safe='')}")["object"]["sha"]
        head_sha = api(f"repos/{repo}/git/ref/heads/{quote(head, safe='')}")["object"]["sha"]
        comparison = api(f"repos/{repo}/compare/{base_sha}...{head_sha}")
        if not comparison["commits"] or len(comparison["commits"]) != comparison["total_commits"]:
            raise ValueError("コミット取得が不完全（空の比較・API の250件上限も確認する）")
        if comparison["commits"][-1]["sha"] != head_sha:
            raise ValueError("コミット一覧と HEAD が一致しない")
        numbers = set(conventions.related_issues(data["body"]))
        for commit in comparison["commits"]:
            message = commit["commit"]["message"]
            errors = conventions.validate_commit(message)
            if errors:
                raise ValueError(f"{commit['sha']}: {'; '.join(errors)}")
            numbers.add(int(conventions.COMMIT_TITLE.fullmatch(message.splitlines()[0])[1]))
        for number in sorted(numbers):
            if "pull_request" in api(f"repos/{repo}/issues/{number}"):
                raise ValueError(f"#{number} は Issue ではなく PR")
        for branch, sha in ((base, base_sha), (head, head_sha)):
            if api(f"repos/{repo}/git/ref/heads/{quote(branch, safe='')}")["object"]["sha"] != sha:
                raise ValueError("検査中にブランチが変わったため最新の内容で再実行する")
    if check_only:
        return ""
    args = ["gh", kind, "create", f"--repo={repo}",
            f"--title={data['title']}", "--body-file=-"]
    args.extend(f"--label={label}" for label in labels)
    if kind == "pr":
        args.extend([f"--head={head}", f"--base={base}", "--draft"])
    return send(args, body=data["body"])


def send(args, *, body):
    """send は検査済み本文を stdin で gh に渡す。自動再試行は行わない。"""
    return subprocess.run(args, input=body, text=True, capture_output=True,
                          encoding="utf-8", check=True, timeout=60).stdout


def main(argv=None):
    parser = argparse.ArgumentParser(description="規約検査に成功してから Issue・Draft PR を作成する")
    commands = parser.add_subparsers(dest="kind", required=True)
    for kind in ("issue", "pr"):
        command = commands.add_parser(kind)
        command.add_argument("--repo", required=True)
        command.add_argument("--title", required=True)
        command.add_argument("--body-file", required=True)
        command.add_argument("--label", action="append", required=True)
        command.add_argument("--check-only", action="store_true")
        if kind == "pr":
            command.add_argument("--head", required=True)
            command.add_argument("--base", default="main")
    args = parser.parse_args(argv)
    try:
        data = {"title": args.title, "body": Path(args.body_file).read_text(encoding="utf-8"),
                "labels": args.label}
        result = create(args.kind, args.repo, data, api=github_api, send=send,
                        head=getattr(args, "head", None), base=getattr(args, "base", "main"),
                        check_only=args.check_only)
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as error:
        print(f"作成前検査失敗または作成失敗: {error}", file=sys.stderr)
        print("作成を自動再試行しない。通信失敗の場合は GitHub 上の作成有無を確認する。", file=sys.stderr)
        return 1
    print("作成前検査が成功した（書き込みなし）" if args.check_only else result.strip())
    return 0


if __name__ == "__main__":
    sys.exit(main())
