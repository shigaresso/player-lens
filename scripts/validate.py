"""規約検査の CLI と Git・GitHub の I/O。外部処理には期限を設ける。"""

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

import conventions


def run(args, *, cwd=None):
    result = subprocess.run(args, cwd=cwd, capture_output=True, text=True,
                            timeout=60, check=True)
    return result.stdout


def resolve_commit(ref, *, cwd=None):
    return run(["git", "rev-parse", "--verify", "--end-of-options", f"{ref}^{{commit}}"], cwd=cwd).strip()


def validate_range(base, head, *, cwd=None):
    """validate_range は base..head の全コミットを検査する。"""
    base = resolve_commit(base, cwd=cwd)
    head = resolve_commit(head, cwd=cwd)
    commits = run(["git", "rev-list", "--reverse", f"{base}..{head}"], cwd=cwd).splitlines()
    validate_commits(commits, cwd=cwd)
    return commits


def validate_commits(commits, *, cwd=None):
    for commit in commits:
        message = run(["git", "show", "--no-patch", "--format=%B", commit], cwd=cwd)
        errors = conventions.validate_commit(message)
        if errors:
            raise ValueError(f"{commit}: {'; '.join(errors)}")


def validate_pre_push(updates, remote, *, cwd=None):
    """validate_pre_push は Git が stdin に渡す全 ref 更新を確認する。"""
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_.-]*", remote):
        raise ValueError("pre-push には登録済み remote の名前を渡す")
    commits = set()
    for line in updates.splitlines():
        fields = line.split()
        if len(fields) != 4:
            raise ValueError("pre-push の更新は ref・SHA・ref・SHA の4項目とする")
        _, local, destination, old = fields
        if not all(re.fullmatch(r"(?:[0-9a-f]{40}|[0-9a-f]{64})", sha) for sha in (local, old)):
            raise ValueError("更新の SHA が不正")
        if destination == "refs/heads/main":
            raise ValueError("main は PR 経由で更新する")
        if not local.strip("0"):
            continue
        local = resolve_commit(local, cwd=cwd)
        if old.strip("0"):
            commits.update(validate_range(old, local, cwd=cwd))
        else:
            refs = run(["git", "for-each-ref", "--format=%(objectname)", f"refs/remotes/{remote}/"], cwd=cwd).splitlines()
            if not refs:
                raise ValueError("新規 ref の検査前に remote の参照を fetch する")
            commits.update(run(["git", "rev-list", "--reverse", local, "--not", *refs], cwd=cwd).splitlines())
    validate_commits(sorted(commits), cwd=cwd)
    return sorted(commits)


def validate_github(event, api):
    """validate_github は現在のメタデータと全コミットを読み取りで確認する。"""
    repo = event["repository"]["full_name"]
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repo):
        raise ValueError("リポジトリ名が不正")
    if "pull_request" not in event:
        number = event["issue"]["number"]
        if not isinstance(number, int) or number < 1:
            raise ValueError("Issue 番号が不正")
        data = api(f"repos/{repo}/issues/{number}")
        if "pull_request" in data:
            raise ValueError("Issue イベントの対象が PR")
        return conventions.validate_issue(data)
    payload = event["pull_request"]
    number = payload["number"]
    if not isinstance(number, int) or number < 1:
        raise ValueError("PR 番号が不正")
    data = api(f"repos/{repo}/pulls/{number}")
    errors = conventions.validate_pr(data)
    if data["head"]["sha"] != payload["head"]["sha"]:
        errors.append("PR の HEAD が変わったため最新の実行で確認する")
    numbers = set(conventions.related_issues(data.get("body") or ""))
    commits = api(f"repos/{repo}/pulls/{number}/commits", paginate=True)
    if not commits or len(commits) != data["commits"]:
        errors.append("PR のコミット取得が不完全（API 上限も確認する）")
    for commit in commits:
        errors.extend(f"{commit['sha']}: {error}" for error in conventions.validate_commit(commit["commit"]["message"]))
        match = conventions.COMMIT_TITLE.fullmatch(commit["commit"]["message"].splitlines()[0])
        if match:
            numbers.add(int(match[1]))
    for linked in sorted(numbers):
        if "pull_request" in api(f"repos/{repo}/issues/{linked}"):
            errors.append(f"#{linked} は Issue ではなく PR")
    return errors


def github_api(endpoint, *, paginate=False):
    """GitHub の読み取りを gh に委譲する。認証情報は出力しない。"""
    args = ["gh", "api", "--method", "GET", endpoint]
    if paginate:
        args.extend(["--paginate", "--slurp"])
    data = json.loads(run(args))
    return [item for page in data for item in page] if paginate else data


def main(argv=None):
    parser = argparse.ArgumentParser(description="PlayerLens の Git・GitHub 規約を検査する")
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("commit").add_argument("file")
    for name in ("pr", "issue"):
        metadata = commands.add_parser(name)
        metadata.add_argument("--title", required=True)
        metadata.add_argument("--body-file", required=True)
        metadata.add_argument("--label", action="append", default=[])
    commits = commands.add_parser("range")
    commits.add_argument("base")
    commits.add_argument("head")
    commands.add_parser("pre-push").add_argument("remote")
    commands.add_parser("github").add_argument("event_file")
    args = parser.parse_args(argv)
    try:
        if args.command == "commit":
            errors = conventions.validate_commit(Path(args.file).read_text(encoding="utf-8"))
        elif args.command in ("pr", "issue"):
            data = {"title": args.title, "body": Path(args.body_file).read_text(encoding="utf-8"), "labels": args.label}
            errors = (conventions.validate_pr if args.command == "pr" else conventions.validate_issue)(data)
        elif args.command == "range":
            validate_range(args.base, args.head)
            errors = []
        elif args.command == "pre-push":
            validate_pre_push(sys.stdin.read(), args.remote)
            errors = []
        else:
            event = json.loads(Path(args.event_file).read_text(encoding="utf-8"))
            errors = validate_github(event, github_api)
        if errors:
            raise ValueError("; ".join(errors))
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as error:
        print(f"規約検査失敗: {error}", file=sys.stderr)
        return 1
    print("規約検査が成功した")
    return 0


if __name__ == "__main__":
    sys.exit(main())
