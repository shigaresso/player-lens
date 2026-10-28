"""固定版 Gitleaks を一時 Git リポジトリで検証する。外部接続は行わない。"""

import contextlib
import io
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import check_secrets


class SecretToolTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory(prefix="player lens secrets ")
        self.addCleanup(directory.cleanup)
        self.repo = Path(directory.name)
        self.git("init", "-q", "--initial-branch=main")
        self.git("config", "user.name", "秘密情報検査")
        self.git("config", "user.email", "secrets@example.invalid")
        self.token = "gh" + "p_" + "Ab3Cd4Ef5Gh6Ij7Kl8Mn9Op0Qr1St2Uv3Wx4"

    def git(self, *args):
        return subprocess.run(["git", *args], cwd=self.repo, text=True,
                              capture_output=True, timeout=30, check=True).stdout.strip()

    def commit(self, text, message=None):
        (self.repo / "source.txt").write_text(text)
        self.git("add", "source.txt")
        self.git("commit", "-qm", message or "(#97) test: 検査用の入力を追加した\n\n- 検査用に記録した")
        return self.git("rev-parse", "HEAD")

    def test_intermediate_commit_is_rejected_after_secret_is_removed(self):
        self.commit("safe\n")
        added = self.commit(self.token + "\n")
        removed = self.commit("safe again\n")
        with self.assertRaises(ValueError) as raised:
            check_secrets.check_commits(self.repo, [added, removed])
        self.assertNotIn(self.token, str(raised.exception))
        self.assertIn("github-pat", str(raised.exception))

    def test_secret_in_commit_message_is_rejected(self):
        commit = self.commit("safe\n", "(#97) test: 検査した\n\n- " + self.token)
        with self.assertRaises(ValueError) as raised:
            check_secrets.check_commits(self.repo, [commit])
        self.assertNotIn(self.token, str(raised.exception))

    def test_message_cli_rejects_secret_without_printing_it(self):
        message = self.repo / "message.txt"
        message.write_text("(#97) test: 検査した\n\n- " + self.token)
        output = io.StringIO()
        with contextlib.redirect_stderr(output):
            result = check_secrets.main(["message", str(message)], repo=self.repo)
        self.assertEqual(1, result)
        self.assertIn("github-pat", output.getvalue())
        self.assertNotIn(self.token, output.getvalue())

    def test_history_cli_detects_removed_secret(self):
        self.commit(self.token + "\n")
        self.commit("safe\n")
        output = io.StringIO()
        with contextlib.redirect_stderr(output):
            result = check_secrets.main(["history"], repo=self.repo)
        self.assertEqual(1, result)
        self.assertNotIn(self.token, output.getvalue())
        self.assertIn("github-pat", output.getvalue())

    def test_pre_push_cli_rejects_secret_in_new_branch_history(self):
        base = self.commit("safe\n")
        self.git("update-ref", "refs/remotes/origin/main", base)
        self.commit(self.token + "\n")
        head = self.commit("safe again\n")
        updates = f"refs/heads/topic {head} refs/heads/topic {'0' * 40}\n"
        output = io.StringIO()
        with patch("sys.stdin", io.StringIO(updates)), contextlib.redirect_stderr(output):
            result = check_secrets.main(["pre-push", "origin"], repo=self.repo)
        self.assertEqual(1, result)
        self.assertNotIn(self.token, output.getvalue())
        self.assertIn("github-pat", output.getvalue())


if __name__ == "__main__":
    unittest.main(verbosity=2)
