import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

import validate
from test_conventions import pr, issue


class GitTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.repo = Path(self.temp.name)
        self.git("init", "-q", "--initial-branch=main")
        self.git("config", "user.email", "test@example.invalid")
        self.git("config", "user.name", "検証用")
        self.base = self.commit("Initial commit")

    def git(self, *args):
        return subprocess.run(["git", *args], cwd=self.repo, check=True,
                              capture_output=True, text=True).stdout.strip()

    def commit(self, message):
        path = self.repo / "message.txt"
        path.write_text(message)
        self.git("commit", "--allow-empty", "-q", "-F", str(path))
        return self.git("rev-parse", "HEAD")

    def test_range_selects_new_commit(self):
        head = self.commit("(#3) ci: 検査を追加した\n\n- 検査を実装した\n")
        self.assertEqual([head], validate.validate_range(self.base, head, cwd=self.repo))


    def test_range_rejects_invalid_commit(self):
        head = self.commit("メッセージ違反")
        with self.assertRaises(ValueError):
            validate.validate_range(self.base, head, cwd=self.repo)

    def test_new_branch_push(self):
        self.git("update-ref", "refs/remotes/origin/main", self.base)
        head = self.commit("(#3) ci: 検査を追加した\n\n- 検査を実装した\n")
        update = f"refs/heads/topic {head} refs/heads/topic {'0' * 40}\n"
        self.assertEqual([head], validate.validate_pre_push(update, "origin", cwd=self.repo))


    def test_deleted_ref_is_not_a_commit(self):
        update = f"(delete) {'0' * 40} refs/heads/topic {self.base}\n"
        self.assertEqual([], validate.validate_pre_push(update, "origin", cwd=self.repo))

    def test_multiple_updates_include_violation(self):
        good = self.commit("(#3) ci: 検査を追加した\n\n- 検査を実装した\n")
        bad = self.commit("規約違反")
        updates = f"refs/heads/a {good} refs/heads/a {self.base}\nrefs/heads/b {bad} refs/heads/b {good}\n"
        with self.assertRaises(ValueError):
            validate.validate_pre_push(updates, "origin", cwd=self.repo)

    def test_unknown_base_is_an_error(self):
        with self.assertRaises(subprocess.CalledProcessError):
            validate.validate_range("missing-ref", self.base, cwd=self.repo)


class GitHubTests(unittest.TestCase):
    def event(self):
        return {"repository": {"full_name": "shigaresso/player-lens"},
                "pull_request": {"number": 4, "head": {"sha": "a" * 40}}}

    def api(self, endpoint, **kwargs):
        if endpoint.endswith("/commits"):
            return [{"sha": "a" * 40, "commit": {"message": "(#3) ci: 検査を追加した\n\n- 検査を実装した\n"}}]
        if "/pulls/" in endpoint:
            return {**pr(), "head": {"sha": "a" * 40}, "commits": 1}
        return issue()

    def test_live_pr_and_commits(self):
        self.assertEqual([], validate.validate_github(self.event(), self.api))


    def test_pr_number_cannot_replace_issue(self):
        def api(endpoint, **kwargs):
            data = self.api(endpoint, **kwargs)
            return {**data, "pull_request": {}} if "/issues/" in endpoint else data
        self.assertTrue(validate.validate_github(self.event(), api))


    def test_incomplete_commit_list(self):
        def api(endpoint, **kwargs):
            return [] if endpoint.endswith("/commits") else self.api(endpoint, **kwargs)
        self.assertTrue(validate.validate_github(self.event(), api))


    def test_stale_head(self):
        event = self.event()
        event["pull_request"]["head"]["sha"] = "b" * 40
        self.assertTrue(validate.validate_github(event, self.api))


    def test_issue_edited_to_completed(self):
        event = {"repository": {"full_name": "shigaresso/player-lens"}, "issue": {"number": 3}}
        self.assertEqual([], validate.validate_github(event, lambda *a, **k: issue(
            issue()["body"].replace("[ ]", "[x]"))))


    def test_api_failure_does_not_pass(self):
        def api(*args, **kwargs):
            raise subprocess.CalledProcessError(1, ["gh", "api"])
        with self.assertRaises(subprocess.CalledProcessError):
            validate.validate_github(self.event(), api)


class CliTests(unittest.TestCase):
    def test_invalid_message_exits_nonzero(self):
        with tempfile.TemporaryDirectory() as folder:
            path = Path(folder) / "message.txt"
            path.write_text("形式違反")
            result = subprocess.run([sys.executable, str(Path(validate.__file__)), "commit", str(path)],
                                    capture_output=True, text=True)
        self.assertEqual(1, result.returncode)
        self.assertIn("タイトル", result.stderr)


if __name__ == "__main__":
    unittest.main()
