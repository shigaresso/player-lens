"""一時リポジトリで実際の Git/Lefthook を検証する。push は実行しない。"""

import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

from git_test_environment import isolated_git_environment

ROOT = Path(__file__).resolve().parents[1]


class HookTests(unittest.TestCase):
    def setUp(self):
        self.enterContext(isolated_git_environment())
        self.temp = tempfile.TemporaryDirectory(prefix="player lens hooks ")
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        for directory in ("scripts", ".github", ".lefthook", "docs"):
            shutil.copytree(ROOT / directory, self.repo / directory, ignore=shutil.ignore_patterns("__pycache__"))
        for filename in ("lefthook.yml", "AGENTS.md", ".gitignore", ".gitleaks.toml"):
            shutil.copy2(ROOT / filename, self.repo / filename)
        (self.repo / ".tools").symlink_to(ROOT / ".tools", target_is_directory=True)
        # フックの接続と失敗伝播を検証する。Docker 自体は別の実資源検査で扱う。
        (self.repo / "scripts/check-app.sh").write_text('''#!/usr/bin/env bash
set -euo pipefail
printf 'called\\n' >> app-check.txt
if [[ "${HOOK_APP_FAIL:-0}" == 1 ]]; then
  echo 'アプリケーション検査の失敗' >&2
  exit 1
fi
''', encoding="utf-8")
        self.env = {**os.environ, "PATH": f"{self.repo / '.tools'}:{os.environ['PATH']}"}
        binary = self.repo / "fake-bin"
        binary.mkdir()
        docker = binary / "docker"
        docker.write_text('''#!/usr/bin/env bash
if [[ "$2" == run && "${HOOK_FORMAT_FAIL:-0}" == 1 ]]; then
  echo '/format-input/example.go'
fi
''')
        docker.chmod(0o755)
        self.env["PATH"] = f"{binary}:{self.env['PATH']}"
        self.command(["git", "init", "-q", "--initial-branch=main"])
        self.command(["git", "config", "user.name", "フック検証"])
        self.command(["git", "config", "user.email", "hooks@example.invalid"])
        self.commit("Initial commit", expected=0)
        self.base = self.command(["git", "rev-parse", "HEAD"]).stdout.strip()
        # 導入前の不正な入力を用意し、導入後の pre-push が拒否することを確認する。
        self.commit("形式違反の入力", expected=0)
        self.bad = self.command(["git", "rev-parse", "HEAD"]).stdout.strip()
        self.command(["lefthook", "install"])

    def command(self, args, *, input=None, expected=0):
        result = subprocess.run(args, cwd=self.repo, env=self.env, input=input,
                                text=True, capture_output=True, timeout=60)
        self.assertEqual(expected, result.returncode, result.stdout + result.stderr)
        return result

    def commit(self, message, expected):
        path = self.repo / "message.txt"
        path.write_text(message, encoding="utf-8")
        return self.command(["git", "commit", "--allow-empty", "-q", "-F", str(path)], expected=expected)

    def test_commit_msg_rejects_violation(self):
        result = self.commit("形式違反", expected=1)
        self.assertIn("タイトル", result.stdout + result.stderr)

    def test_commit_accepts_valid_message(self):
        self.commit("(#3) ci: 検査を追加した\n\n- 検査を実装した\n", expected=0)

    def test_pre_commit_rejects_staged_whitespace(self):
        (self.repo / "example.txt").write_text("末尾空白  \n", encoding="utf-8")
        self.command(["git", "add", "example.txt"])
        result = self.commit("(#3) ci: 検査を追加した\n\n- 検査を実装した\n", expected=1)
        self.assertIn("trailing whitespace", result.stdout + result.stderr)

    def test_pre_commit_rejects_staged_go_format(self):
        source = self.repo / "example.go"
        staged = "package example;var Value=1\n"
        source.write_text(staged)
        self.command(["git", "add", "example.go"])
        source.write_text("package example\n\nvar Value = 1\n")
        self.env["HOOK_FORMAT_FAIL"] = "1"
        result = self.commit("(#13) test: 整形検査を確認した\n\n- 検査を確認した\n", expected=1)
        self.assertIn("example.go", result.stdout + result.stderr)
        self.assertEqual(staged, self.command(["git", "show", ":example.go"]).stdout)

    def test_pre_commit_accepts_staged_go_format(self):
        source = self.repo / "example.go"
        staged = "package example\n\nvar Value = 1\n"
        source.write_text(staged)
        self.command(["git", "add", "example.go"])
        source.write_text("package example;var Value=2\n")
        self.commit("(#13) test: 整形検査を確認した\n\n- 検査を確認した\n", expected=0)
        self.assertEqual(staged, self.command(["git", "show", "HEAD:example.go"]).stdout)
        self.assertEqual("package example;var Value=2\n", source.read_text())

    def test_pre_push_receives_stdin_and_rejects(self):
        updates = f"refs/heads/topic {self.bad} refs/heads/topic {self.base}\n"
        result = self.command([str(self.repo / ".git/hooks/pre-push"), "origin", "unused"],
                              input=updates, expected=1)
        self.assertIn("タイトル", result.stdout + result.stderr)

    def test_pre_push_accepts_new_branch(self):
        self.command(["git", "update-ref", "refs/remotes/origin/main", self.bad])
        self.commit("(#3) ci: 検査を追加した\n\n- 検査を実装した\n", expected=0)
        head = self.command(["git", "rev-parse", "HEAD"]).stdout.strip()
        updates = f"refs/heads/topic {head} refs/heads/topic {'0' * 40}\n"
        self.command([str(self.repo / ".git/hooks/pre-push"), "origin", "unused"], input=updates)
        self.assertEqual("called\n", (self.repo / "app-check.txt").read_text())

    def test_pre_push_rejects_application_failure(self):
        self.command(["git", "update-ref", "refs/remotes/origin/main", self.bad])
        self.commit("(#3) ci: 検査を追加した\n\n- 検査を実装した\n", expected=0)
        head = self.command(["git", "rev-parse", "HEAD"]).stdout.strip()
        self.env["HOOK_APP_FAIL"] = "1"
        updates = f"refs/heads/topic {head} refs/heads/topic {'0' * 40}\n"
        result = self.command([str(self.repo / ".git/hooks/pre-push"), "origin", "unused"],
                              input=updates, expected=1)
        self.assertIn("アプリケーション検査の失敗", result.stdout + result.stderr)

    def test_pre_commit_rejects_staged_secret_after_worktree_is_cleaned(self):
        token = "gh" + "p_" + "Ab3Cd4Ef5Gh6Ij7Kl8Mn9Op0Qr1St2Uv3Wx4"
        path = self.repo / "secret.txt"
        path.write_text(token)
        self.command(["git", "add", path.name])
        path.write_text("safe")
        result = self.commit("(#97) test: 検査を確認した\n\n- 検査した\n", expected=1)
        self.assertIn("github-pat", result.stdout + result.stderr)
        self.assertNotIn(token, result.stdout + result.stderr)
        self.assertEqual(token, self.command(["git", "show", ":secret.txt"]).stdout)
        self.assertEqual("safe", path.read_text())

    def test_commit_msg_rejects_secret(self):
        token = "gh" + "p_" + "Ab3Cd4Ef5Gh6Ij7Kl8Mn9Op0Qr1St2Uv3Wx4"
        result = self.commit("(#97) test: 検査を確認した\n\n- " + token, expected=1)
        self.assertIn("github-pat", result.stdout + result.stderr)
        self.assertNotIn(token, result.stdout + result.stderr)

    def test_pre_push_stops_before_application_on_secret_check_failure(self):
        self.command(["git", "update-ref", "refs/remotes/origin/main", self.bad])
        self.commit("(#97) test: 検査を確認した\n\n- 検査した\n", expected=0)
        head = self.command(["git", "rev-parse", "HEAD"]).stdout.strip()
        # 送信履歴の検査自体は verify_secrets.py の実ツール検証が担当する。
        checker = self.repo / "scripts/check_secrets.py"
        checker.write_text(checker.read_text().replace(
            "sys.exit(main())",
            'print("秘密情報検査の失敗", file=sys.stderr); sys.exit(1)'))
        updates = f"refs/heads/topic {head} refs/heads/topic {'0' * 40}\n"
        result = self.command([str(self.repo / ".git/hooks/pre-push"), "origin", "unused"],
                              input=updates, expected=1)
        self.assertIn("秘密情報検査の失敗", result.stdout + result.stderr)
        self.assertFalse((self.repo / "app-check.txt").exists())


if __name__ == "__main__":
    unittest.main(verbosity=2)
