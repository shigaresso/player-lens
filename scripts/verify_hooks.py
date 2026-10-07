"""一時リポジトリで実際の Git/Lefthook を検証する。push は実行しない。"""

import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class HookTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="player lens hooks ")
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        for directory in ("scripts", ".github", ".lefthook", "docs"):
            shutil.copytree(ROOT / directory, self.repo / directory, ignore=shutil.ignore_patterns("__pycache__"))
        for filename in ("lefthook.yml", "AGENTS.md", ".gitignore"):
            shutil.copy2(ROOT / filename, self.repo / filename)
        (self.repo / ".tools").symlink_to(ROOT / ".tools", target_is_directory=True)
        self.env = {**os.environ, "PATH": f"{self.repo / '.tools'}:{os.environ['PATH']}"}
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


if __name__ == "__main__":
    unittest.main(verbosity=2)
