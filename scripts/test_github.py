import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path
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

    def test_check_only_validates_without_creation(self):
        self.assertEqual('', self.create('issue', issue(), check_only=True))
        self.api.assert_called_once_with('repos/example/project/labels', paginate=True)
        self.send.assert_not_called()

    def test_comma_label_is_not_reinterpreted_as_multiple_labels(self):
        data = issue()
        data['labels'] = ['ci,testing']
        self.api.return_value = [{'name': 'ci,testing'}]
        with self.assertRaisesRegex(ValueError, 'カンマ'):
            self.create('issue', data)
        self.send.assert_not_called()


    def test_invalid_pr_body_stops_before_api(self):
        with self.assertRaises(ValueError):
            self.create('pr', pr('本文なし'), head='codex/3-check')
        self.api.assert_not_called()
        self.send.assert_not_called()


    def test_api_failure_stops_creation(self):
        self.api.side_effect = subprocess.CalledProcessError(1, ['gh', 'api'])
        with self.assertRaises(subprocess.CalledProcessError):
            self.create('issue', issue())
        self.send.assert_not_called()


    def test_missing_remote_branch_stops_creation(self):
        def api(endpoint, **kwargs):
            if '/git/ref/heads/' in endpoint and not endpoint.endswith('/main'):
                raise subprocess.CalledProcessError(1, ['gh', 'api'])
            return self.remote_api(endpoint, **kwargs)
        self.api.side_effect = api
        with self.assertRaises(subprocess.CalledProcessError):
            self.create('pr', pr(), head='codex/3-check')
        self.send.assert_not_called()


    def test_empty_comparison_stops_creation(self):
        def api(endpoint, **kwargs):
            return {'total_commits': 0, 'commits': []} if '/compare/' in endpoint else self.remote_api(endpoint, **kwargs)
        self.api.side_effect = api
        with self.assertRaises(ValueError):
            self.create('pr', pr(), head='codex/3-check')
        self.send.assert_not_called()


    def test_commit_issue_is_checked_even_when_not_in_pr_body(self):
        def api(endpoint, **kwargs):
            data = self.remote_api(endpoint, **kwargs)
            if '/compare/' in endpoint:
                data['commits'][0]['commit']['message'] = '(#7) ci: 検査を追加した\n\n- 実装した\n'
            if endpoint.endswith('/issues/7'):
                return {'pull_request': {}}
            return data
        self.api.side_effect = api
        with self.assertRaises(ValueError):
            self.create('pr', pr(), head='codex/3-check')
        self.api.assert_any_call('repos/example/project/issues/7')
        self.send.assert_not_called()


    def test_creation_failure_is_not_retried(self):
        self.send.side_effect = subprocess.TimeoutExpired(['gh', 'issue', 'create'], 60)
        with self.assertRaises(subprocess.TimeoutExpired):
            self.create('issue', issue())
        self.send.assert_called_once()


class CliTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='player lens github ')
        self.addCleanup(self.temp.cleanup)
        self.folder = Path(self.temp.name)
        self.body = self.folder / 'body.md'
        self.body.write_text(issue()['body'])
        self.record = self.folder / 'calls.jsonl'
        fake = self.folder / 'gh'
        fake.write_text(f'#!{sys.executable}\n' + '''import json, os, sys
body = sys.stdin.read() if '--body-file=-' in sys.argv else None
with open(os.environ['GH_TEST_RECORD'], 'a') as file:
    file.write(json.dumps({'args': sys.argv[1:], 'body': body}) + '\\n')
if sys.argv[1] == 'api':
    endpoint = sys.argv[4]
    if '/labels' in endpoint:
        result = [[{'name': 'ci'}]]
    elif '/git/ref/heads/' in endpoint:
        result = {'object': {'sha': 'a' * 40 if endpoint.endswith('/main') else 'b' * 40}}
    elif '/compare/' in endpoint:
        result = {'total_commits': 1, 'commits': [{'sha': 'b' * 40, 'commit': {'message': '(#3) ci: 検査を追加した\\n\\n- 実装した\\n'}}]}
    else:
        result = {'number': 3}
    print(json.dumps(result))
else:
    print('https://github.com/example/project/' + ('pull/4' if sys.argv[1] == 'pr' else 'issues/4'))
''')
        fake.chmod(0o755)
        self.env = {**os.environ, 'PATH': f'{self.folder}:{os.environ["PATH"]}',
                    'GH_TEST_RECORD': str(self.record)}

    def cli(self, *extra, kind='issue'):
        return subprocess.run([sys.executable, str(Path(github.__file__)), kind,
                               '--repo', 'example/project', '--title=検査を追加する $(echo test)',
                               '--body-file', str(self.body), '--label', 'ci', *extra],
                              env=self.env, capture_output=True, text=True, timeout=10)

    def test_cli_sends_exact_input_to_gh_without_shell(self):
        result = self.cli()
        self.assertEqual(0, result.returncode, result.stderr)
        calls = [json.loads(line) for line in self.record.read_text().splitlines()]
        self.assertEqual(2, len(calls))
        self.assertEqual(self.body.read_text(), calls[1]['body'])
        self.assertIn('--title=検査を追加する $(echo test)', calls[1]['args'])
        self.assertIn('https://github.com/example/project/issues/4', result.stdout)

    def test_cli_reports_validation_failure_without_creating(self):
        self.body.write_text('本文なし')
        result = self.cli()
        self.assertEqual(1, result.returncode)
        self.assertIn('作成前検査失敗', result.stderr)
        self.assertNotIn('Traceback', result.stderr)
        self.assertFalse(self.record.exists())

    def test_cli_check_only_calls_read_api_without_creation(self):
        result = self.cli('--check-only')
        self.assertEqual(0, result.returncode, result.stderr)
        calls = [json.loads(line) for line in self.record.read_text().splitlines()]
        self.assertEqual(1, len(calls))
        self.assertEqual(['api', '--method', 'GET'], calls[0]['args'][:3])
        self.assertIn('書き込みなし', result.stdout)

    def test_cli_pr_checks_remote_then_creates_draft_without_pushing(self):
        self.body.write_text(pr()['body'])
        result = self.cli('--head=codex/3-check', kind='pr')
        self.assertEqual(0, result.returncode, result.stderr)
        calls = [json.loads(line) for line in self.record.read_text().splitlines()]
        self.assertEqual(8, len(calls))
        for call in calls[:-1]:
            self.assertEqual(['api', '--method', 'GET'], call['args'][:3])
        self.assertEqual(['pr', 'create'], calls[-1]['args'][:2])
        self.assertIn('--head=codex/3-check', calls[-1]['args'])
        self.assertIn('--base=main', calls[-1]['args'])
        self.assertIn('--draft', calls[-1]['args'])
        self.assertEqual(self.body.read_text(), calls[-1]['body'])
        self.assertIn('https://github.com/example/project/pull/4', result.stdout)

    def test_cli_pr_check_only_does_not_create(self):
        self.body.write_text(pr()['body'])
        result = self.cli('--head=codex/3-check', '--check-only', kind='pr')
        self.assertEqual(0, result.returncode, result.stderr)
        calls = [json.loads(line) for line in self.record.read_text().splitlines()]
        self.assertEqual(7, len(calls))
        self.assertTrue(all(call['args'][:3] == ['api', '--method', 'GET'] for call in calls))


if __name__ == "__main__":
    unittest.main()
