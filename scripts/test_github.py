import unittest
from unittest.mock import Mock

import github
from test_conventions import issue, pr


class CreationTests(unittest.TestCase):
    def setUp(self):
        self.api = Mock(return_value=[{"name": "ci"}])
        self.send = Mock(return_value="https://github.com/example/project/issues/4\n")

    def create(self, kind, data, **kwargs):
        return github.create(kind, "example/project", data,
                             api=self.api, send=self.send, **kwargs)

    def test_valid_issue_sends_the_validated_body(self):
        data = issue()
        self.assertEqual(self.send.return_value, self.create("issue", data))
        self.send.assert_called_once_with(
            ["gh", "issue", "create", "--repo=example/project",
             f"--title={data['title']}", "--body-file=-", "--label=ci"],
            body=data["body"])

    def test_invalid_issue_stops_before_api_or_creation(self):
        with self.assertRaises(ValueError):
            self.create("issue", issue("空欄"))
        self.api.assert_not_called()
        self.send.assert_not_called()

    def test_missing_label_stops_creation(self):
        self.api.return_value = []
        with self.assertRaisesRegex(ValueError, "ラベル"):
            self.create("issue", issue())
        self.send.assert_not_called()

    def remote_api(self, endpoint, **kwargs):
        if endpoint.endswith('/labels'):
            return [{"name": "ci"}]
        if '/git/ref/heads/' in endpoint:
            return {"object": {"sha": "a" * 40 if endpoint.endswith('/main') else "b" * 40}}
        if '/compare/' in endpoint:
            return {"total_commits": 1, "commits": [
                {"sha": "b" * 40, "commit": {"message": "(#3) ci: 検査を追加した\n\n- 実装した\n"}}]}
        return {"number": 3}

    def test_valid_pr_checks_remote_and_creates_explicit_draft(self):
        self.api.side_effect = self.remote_api
        data = pr()
        self.create("pr", data, head="codex/3-check", base="main")
        self.api.assert_any_call(f"repos/example/project/compare/{'a' * 40}...{'b' * 40}")
        self.send.assert_called_once_with(
            ["gh", "pr", "create", "--repo=example/project",
             f"--title={data['title']}", "--body-file=-", "--label=ci",
             "--head=codex/3-check", "--base=main", "--draft"], body=data["body"])

    def test_invalid_remote_commit_stops_creation(self):
        def api(endpoint, **kwargs):
            data = self.remote_api(endpoint, **kwargs)
            if '/compare/' in endpoint:
                data['commits'][0]['commit']['message'] = '本文なし'
            return data
        self.api.side_effect = api
        with self.assertRaises(ValueError):
            self.create("pr", pr(), head="codex/3-check")
        self.send.assert_not_called()

    def test_incomplete_remote_commits_stop_creation(self):
        def api(endpoint, **kwargs):
            data = self.remote_api(endpoint, **kwargs)
            if '/compare/' in endpoint:
                data['total_commits'] = 251
            return data
        self.api.side_effect = api
        with self.assertRaisesRegex(ValueError, "不完全"):
            self.create("pr", pr(), head="codex/3-check")
        self.send.assert_not_called()

    def test_commit_head_mismatch_stops_creation(self):
        def api(endpoint, **kwargs):
            data = self.remote_api(endpoint, **kwargs)
            if '/compare/' in endpoint:
                data['commits'][-1]['sha'] = 'c' * 40
            return data
        self.api.side_effect = api
        with self.assertRaisesRegex(ValueError, "HEAD"):
            self.create("pr", pr(), head="codex/3-check")
        self.send.assert_not_called()

    def test_linked_pr_number_stops_creation(self):
        def api(endpoint, **kwargs):
            return {"pull_request": {}} if '/issues/' in endpoint else self.remote_api(endpoint, **kwargs)
        self.api.side_effect = api
        with self.assertRaisesRegex(ValueError, "Issue"):
            self.create("pr", pr(), head="codex/3-check")
        self.send.assert_not_called()

    def test_changed_remote_branch_stops_creation(self):
        calls = 0
        def api(endpoint, **kwargs):
            nonlocal calls
            data = self.remote_api(endpoint, **kwargs)
            if '/git/ref/heads/' in endpoint and not endpoint.endswith('/main'):
                calls += 1
                if calls > 1:
                    data['object']['sha'] = 'c' * 40
            return data
        self.api.side_effect = api
        with self.assertRaisesRegex(ValueError, "変わった"):
            self.create("pr", pr(), head="codex/3-check")
        self.send.assert_not_called()

    def test_invalid_repo_stops_before_api(self):
        with self.assertRaises(ValueError):
            github.create('issue', 'example/project?other', issue(), api=self.api, send=self.send)
        self.api.assert_not_called()
        self.send.assert_not_called()

    def test_body_snapshot_does_not_change_after_validation(self):
        data = issue()
        original = data['body']
        def api(*args, **kwargs):
            data['body'] = '検査後に空欄へ変更'
            return [{"name": "ci"}]
        self.api.side_effect = api
        self.create('issue', data)
        self.assertEqual(original, self.send.call_args.kwargs['body'])


if __name__ == "__main__":
    unittest.main()
