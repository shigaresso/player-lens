import copy
import json
import os
import subprocess
import sys
import unittest
from pathlib import Path
from unittest.mock import Mock

import validate
import test_validate


REPO = 'example/project'
GOOD = '(#9) ci: 検査を追加した\n\n- 検査を実装した\n'


class MainRangeTests(unittest.TestCase):
    git = test_validate.GitTests.git
    commit = test_validate.GitTests.commit

    def setUp(self):
        test_validate.GitTests.setUp(self)
        self.git('switch', '-q', '-c', 'topic')
        self.topic = self.commit(GOOD)
        self.git('switch', '-q', 'main')
        self.main = self.commit(GOOD.replace('検査を追加した', 'main の変更を追加した'))
        self.git('merge', '--no-ff', '-m', 'Merge pull request #3 from topic', 'topic')
        self.merge = self.git('rev-parse', 'HEAD')
        self.pulls = [{'number': 3, 'merge_commit_sha': self.merge}]
        self.pull = {'merged': True, 'merge_commit_sha': self.merge,
                     'base': {'ref': 'main', 'repo': {'full_name': REPO}},
                     'head': {'sha': self.topic}}
        self.api = Mock(side_effect=self.read)

    def read(self, endpoint, **kwargs):
        if endpoint == f'repos/{REPO}/commits/{self.merge}/pulls':
            return self.pulls
        if endpoint == f'repos/{REPO}/pulls/3':
            return self.pull
        raise AssertionError(endpoint)

    def check(self):
        return validate.validate_main_range(self.base, self.merge, REPO, self.api, cwd=self.repo)

    def test_confirmed_pr_merge_exempts_only_integration_message(self):
        commits = self.check()
        self.assertEqual({self.topic, self.main, self.merge}, set(commits))
        self.api.assert_any_call(f'repos/{REPO}/commits/{self.merge}/pulls', paginate=True)
        self.api.assert_any_call(f'repos/{REPO}/pulls/3')

    def test_unconfirmed_pr_conditions_do_not_exempt_message(self):
        original = copy.deepcopy(self.pull)
        for name, path, value in (
            ('未マージ PR を拒否する', ('merged',), False),
            ('別の統合 SHA を拒否する', ('merge_commit_sha',), 'a' * 40),
            ('別ブランチを拒否する', ('base', 'ref'), 'other'),
            ('別リポジトリを拒否する', ('base', 'repo', 'full_name'), 'other/project'),
            ('第2親と異なる head を拒否する', ('head', 'sha'), self.main),
        ):
            with self.subTest(name=name):
                self.pull = copy.deepcopy(original)
                target = self.pull
                for key in path[:-1]:
                    target = target[key]
                target[path[-1]] = value
                with self.assertRaises(ValueError):
                    self.check()

    def test_missing_or_ambiguous_association_keeps_strict_validation(self):
        for name, pulls in (
            ('関連 PR がなければ拒否する', []),
            ('別 SHA の関連だけでは拒否する', [{'number': 3, 'merge_commit_sha': self.topic}]),
            ('対応 PR が重複すれば拒否する', self.pulls * 2),
        ):
            with self.subTest(name=name):
                self.pulls = pulls
                with self.assertRaises(ValueError):
                    self.check()

    def test_invalid_work_commit_is_not_hidden_by_valid_merge(self):
        self.base = self.merge
        self.git('switch', '-q', 'topic')
        self.topic = self.commit('本文なしの作業コミット')
        self.git('switch', '-q', 'main')
        self.git('merge', '--no-ff', '-m', 'Merge pull request #3 from topic', 'topic')
        self.merge = self.git('rev-parse', 'HEAD')
        self.pull.update(merge_commit_sha=self.merge, head={'sha': self.topic})
        self.pulls = [{'number': 3, 'merge_commit_sha': self.merge}]
        with self.assertRaisesRegex(ValueError, self.topic):
            self.check()

    def test_single_parent_message_is_never_exempted(self):
        head = self.commit('Merge pull request #3 from topic')
        with self.assertRaises(ValueError):
            validate.validate_main_range(self.merge, head, REPO, self.api, cwd=self.repo)
        self.api.assert_not_called()

    def test_api_failure_does_not_allow_merge(self):
        self.api.side_effect = subprocess.CalledProcessError(1, ['gh', 'api'])
        with self.assertRaises(subprocess.CalledProcessError):
            self.check()

    def test_valid_work_range_does_not_need_api(self):
        self.assertEqual([self.topic], validate.validate_main_range(
            self.base, self.topic, REPO, self.api, cwd=self.repo))
        self.api.assert_not_called()

    def test_regular_range_remains_strict(self):
        with self.assertRaisesRegex(ValueError, self.merge):
            validate.validate_range(self.base, self.merge, cwd=self.repo)

    def prepare_cli(self):
        self.responses = self.repo / 'responses.json'
        self.responses.write_text(json.dumps({
            f'repos/{REPO}/commits/{self.merge}/pulls': self.pulls,
            f'repos/{REPO}/pulls/3': self.pull,
        }), encoding='utf-8')
        self.record = self.repo / 'calls.jsonl'
        fake = self.repo / 'gh'
        fake.write_text(f'#!{sys.executable}\n' + '''import json, os, sys
from pathlib import Path
with open(os.environ['MAIN_RANGE_RECORD'], 'a') as file:
    file.write(json.dumps(sys.argv[1:]) + '\\n')
if os.environ.get('MAIN_RANGE_API_FAIL'):
    sys.exit(1)
responses = json.loads(Path(os.environ['MAIN_RANGE_RESPONSES']).read_text())
data = responses[sys.argv[4]]
print(json.dumps([data] if '--paginate' in sys.argv else data))
''', encoding='utf-8')
        fake.chmod(0o755)
        self.env = {**os.environ, 'PATH': f'{self.repo}:{os.environ["PATH"]}',
                    'MAIN_RANGE_RECORD': str(self.record), 'MAIN_RANGE_RESPONSES': str(self.responses)}

    def cli(self, mode='main-range'):
        args = [sys.executable, validate.__file__, mode, self.base, self.merge]
        if mode == 'main-range':
            args.extend(['--repo', REPO])
        return subprocess.run(args, cwd=self.repo, env=self.env,
                              capture_output=True, text=True, timeout=10)

    def test_cli_allows_confirmed_merge_using_read_only_api(self):
        self.prepare_cli()
        result = self.cli()
        self.assertEqual(0, result.returncode, result.stderr)
        calls = [json.loads(line) for line in self.record.read_text().splitlines()]
        self.assertEqual(2, len(calls))
        self.assertTrue(all(call[:3] == ['api', '--method', 'GET'] for call in calls))

    def test_cli_unconfirmed_merge_is_reported_as_failure(self):
        self.pull['merged'] = False
        self.prepare_cli()
        result = self.cli()
        self.assertEqual(1, result.returncode)
        self.assertIn('規約検査失敗', result.stderr)
        self.assertNotIn('Traceback', result.stderr)

    def test_cli_api_failure_is_not_success(self):
        self.prepare_cli()
        self.env['MAIN_RANGE_API_FAIL'] = '1'
        result = self.cli()
        self.assertEqual(1, result.returncode)
        self.assertIn('規約検査失敗', result.stderr)
        self.assertNotIn('Traceback', result.stderr)

    def test_cli_regular_range_has_no_exception_or_network(self):
        self.prepare_cli()
        result = self.cli('range')
        self.assertEqual(1, result.returncode)
        self.assertFalse(self.record.exists())

    def test_valid_merge_message_needs_no_exemption(self):
        self.git('commit', '--amend', '-q', '-m', GOOD)
        self.merge = self.git('rev-parse', 'HEAD')
        self.check()
        self.api.assert_not_called()

    def test_three_parent_merge_is_not_exempted(self):
        tree = self.git('rev-parse', f'{self.merge}^{{tree}}')
        head = self.git('commit-tree', tree, '-p', self.main, '-p', self.topic,
                        '-p', self.base, '-m', 'Merge pull request #3 from topic')
        with self.assertRaises(ValueError):
            validate.validate_main_range(self.base, head, REPO, self.api, cwd=self.repo)
        self.api.assert_not_called()

    def test_invalid_repository_stops_before_api(self):
        with self.assertRaises(ValueError):
            validate.validate_main_range(self.base, self.merge, REPO + '?other',
                                         self.api, cwd=self.repo)
        self.api.assert_not_called()


if __name__ == '__main__':
    unittest.main()
