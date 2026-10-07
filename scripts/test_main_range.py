import copy
import subprocess
import unittest
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


if __name__ == '__main__':
    unittest.main()
