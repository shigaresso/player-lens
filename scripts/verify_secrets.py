"""固定版 Gitleaks を一時 Git リポジトリで検証する。外部接続は行わない。"""

import contextlib
import io
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import check_secrets
from git_test_environment import isolated_git_environment


class SecretToolTests(unittest.TestCase):
    def setUp(self):
        self.enterContext(isolated_git_environment())
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

    def test_pre_push_rebased_history_rejects_removed_secret(self):
        self.commit("base\n")
        self.git("switch", "-qc", "topic")
        old = self.commit("topic\n")
        self.git("update-ref", "refs/remotes/origin/topic", old)
        self.git("switch", "-q", "main")
        (self.repo / "main.txt").write_text("main の更新\n")
        self.git("add", "main.txt")
        self.git("commit", "-qm", "(#145) test: main を更新した\n\n- 入力を追加した")
        self.git("update-ref", "refs/remotes/origin/main", self.git("rev-parse", "HEAD"))
        self.git("switch", "-q", "topic")
        self.git("rebase", "main")
        self.commit(self.token + "\n")
        head = self.commit("safe again\n")
        updates = f"refs/heads/topic {head} refs/heads/topic {old}\n"
        output = io.StringIO()
        with patch("sys.stdin", io.StringIO(updates)), contextlib.redirect_stderr(output):
            result = check_secrets.main(["pre-push", "origin"], repo=self.repo)
        self.assertEqual(1, result)
        self.assertIn("github-pat", output.getvalue())
        self.assertNotIn(self.token, output.getvalue())

    def test_staged_initial_and_partial_inputs_are_preserved(self):
        cases = [
            ("ステージ済みだけに秘密情報があれば拒否する", self.token, "safe", 1),
            ("未ステージだけに秘密情報があっても送信対象に含めない", "safe", self.token, 0),
        ]
        for name, staged, unstaged, expected in cases:
            with self.subTest(name=name), tempfile.TemporaryDirectory() as directory:
                self.repo = Path(directory)
                self.git("init", "-q", "--initial-branch=main")
                path = self.repo / "source with spaces.txt"
                path.write_text(staged)
                self.git("add", path.name)
                path.write_text(unstaged)
                output = io.StringIO()
                with contextlib.redirect_stderr(output), contextlib.redirect_stdout(output):
                    result = check_secrets.main(["staged"], repo=self.repo)
                self.assertEqual(expected, result)
                self.assertEqual(staged, self.git("show", f":{path.name}"))
                self.assertEqual(unstaged, path.read_text())
                self.assertNotIn(self.token, output.getvalue())

    def test_merge_resolution_is_checked_even_when_parents_are_clean(self):
        self.commit("base\n")
        self.git("checkout", "-qb", "side")
        self.commit("side\n")
        self.git("checkout", "-q", "main")
        self.commit("main\n")
        result = subprocess.run(["git", "merge", "side"], cwd=self.repo,
                                text=True, capture_output=True, timeout=30)
        self.assertEqual(1, result.returncode)
        merged = self.commit(self.token + "\n")
        with self.assertRaises(ValueError) as raised:
            check_secrets.check_commits(self.repo, [merged])
        self.assertNotIn(self.token, str(raised.exception))

    def test_clean_history_and_empty_or_deleted_push_are_accepted(self):
        self.commit("safe\n")
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            self.assertEqual(0, check_secrets.main(["history"], repo=self.repo))
            for name, updates in [
                    ("空の更新", ""),
                    ("作業ブランチの削除", f"(delete) {'0' * 40} refs/heads/topic {'a' * 40}\n")]:
                with self.subTest(name=name), patch("sys.stdin", io.StringIO(updates)):
                    self.assertEqual(0, check_secrets.main(["pre-push", "origin"], repo=self.repo))


if __name__ == "__main__":
    unittest.main(verbosity=2)
