"""固定版 Gitleaks を実行し、検出値を出力せずに失敗を伝える。"""

import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import tempfile

import validate

ROOT = Path(__file__).resolve().parents[1]
VERSION = "8.30.1"


class SecretCheckError(ValueError):
    """実値を含まない、利用者へ表示できる検査結果。"""


def scan(repo, mode, *, staged=False, log_opts=None, data=None, run=subprocess.run):
    """scan は標準ルールを実行し、検出・実行異常のいずれでも失敗する。"""
    binary = ROOT / ".tools/gitleaks"
    env = {key: value for key, value in os.environ.items()
           if not key.startswith("GITLEAKS_")}
    version = run([str(binary), "version"], cwd=repo, env=env,
                  capture_output=True, text=True, timeout=30)
    if version.returncode or version.stdout.strip() != VERSION:
        raise SecretCheckError("固定版 Gitleaks が必要。bash scripts/install-tools.sh を実行する")
    with tempfile.TemporaryDirectory(prefix="player-lens-secrets-") as directory:
        report = Path(directory) / "report.json"
        args = [str(binary), mode, "--config", str(ROOT / ".gitleaks.toml"),
                "--redact=100", "--no-banner", "--no-color", "--log-level=error",
                "--ignore-gitleaks-allow", "--gitleaks-ignore-path", os.devnull,
                "--exit-code=3", "--report-format=json", "--report-path", str(report)]
        if staged:
            args.append("--staged")
        if log_opts is not None:
            args.append(f"--log-opts={log_opts}")
        if mode == "git":
            args.append(str(repo))
        result = run(args, cwd=repo, env=env, input=data,
                     capture_output=True, text=True, timeout=120)
        if result.returncode not in (0, 3):
            raise SecretCheckError("Gitleaks の実行に失敗した（診断の実値は表示しない）")
        findings = json.loads(report.read_text())
        if not isinstance(findings, list) or any(
                not isinstance(finding, dict)
                or not isinstance(finding.get("RuleID"), str)
                or not isinstance(finding.get("File"), str)
                or not isinstance(finding.get("StartLine"), int)
                or finding["StartLine"] < 0 for finding in findings):
            raise SecretCheckError("Gitleaks の結果形式が不正")
        if result.returncode == 3 or findings:
            locations = []
            for finding in findings:
                rule = finding.get("RuleID", "")
                if not re.fullmatch(r"[a-z0-9-]{1,100}", rule):
                    rule = "unknown-rule"
                path = Path(finding.get("File") or "入力")
                if path.is_absolute():
                    try:
                        path = path.relative_to(repo)
                    except ValueError:
                        path = Path("入力")
                locations.append(f"{rule}: {json.dumps(str(path), ensure_ascii=False)}:{int(finding.get('StartLine', 0))}")
            raise SecretCheckError("秘密情報の候補を検出した（実値非表示）: " + "; ".join(locations))


def check_commits(repo, commits):
    """check_commits は指定した全コミットをマージ解決の差分も含めて検査する。"""
    resolved = [validate.resolve_commit(commit, cwd=repo) for commit in commits]
    for start in range(0, len(resolved), 100):
        batch = resolved[start:start + 100]
        scan(repo, "git", log_opts="--no-walk --diff-merges=first-parent " + " ".join(batch))
        messages = validate.run(["git", "log", "--no-walk", "--format=%B", *batch], cwd=repo)
        scan(repo, "stdin", data=messages)


def main(argv=None, *, repo=ROOT):
    parser = argparse.ArgumentParser(description="秘密情報を実値非表示で検査する")
    commands = parser.add_subparsers(dest="mode", required=True)
    commands.add_parser("staged")
    commands.add_parser("history")
    commands.add_parser("pre-push").add_argument("remote")
    commands.add_parser("message").add_argument("file")
    args = parser.parse_args(argv)
    try:
        if args.mode == "staged":
            scan(repo, "git", staged=True)
        elif args.mode == "pre-push":
            try:
                commits = validate.validate_pre_push(sys.stdin.read(), args.remote, cwd=repo)
            except ValueError as error:
                # 既存バリデーターは内容の実値を含まない規約違反だけを返す。
                raise SecretCheckError(str(error)) from None
            check_commits(repo, commits)
        elif args.mode == "history":
            commits = validate.run(["git", "rev-list", "--all", "HEAD"], cwd=repo).splitlines()
            check_commits(repo, commits)
        elif args.mode == "message":
            scan(repo, "stdin", data=Path(args.file).read_text())
    except SecretCheckError as error:
        print(f"秘密情報検査失敗: {error}", file=sys.stderr)
        return 1
    except (OSError, ValueError, TypeError, subprocess.SubprocessError):
        print("秘密情報検査失敗: ツール・Git・検査結果を確認する（診断の実値は表示しない）", file=sys.stderr)
        return 1
    print("秘密情報検査が成功した")
    return 0


if __name__ == "__main__":
    sys.exit(main())
