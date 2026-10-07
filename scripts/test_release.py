import unittest
import copy
import subprocess
from unittest.mock import Mock

import release


TITLE = '開発基盤を公開した'
BODY = '\n\n'.join(f'## {name}\n\n内容を確認した。' for name in
                   ('変更内容', '利用方法', '依存環境', '制約', '検証'))
SHA = 'a' * 40
REPO = 'example/project'
TAG = 'v0.0.1'


class InputTests(unittest.TestCase):
    def test_valid_input(self):
        release.validate_input(REPO, TAG, SHA, TITLE, BODY)

    def test_invalid_tag_is_rejected(self):
        with self.assertRaisesRegex(ValueError, 'タグ'):
            release.validate_input(REPO, 'v00.0.1', SHA, TITLE, BODY)

    def test_short_target_is_rejected(self):
        with self.assertRaisesRegex(ValueError, 'SHA'):
            release.validate_input(REPO, TAG, SHA[:7], TITLE, BODY)

    def test_invalid_repo_is_rejected(self):
        with self.assertRaisesRegex(ValueError, 'リポジトリ'):
            release.validate_input(REPO + '?query', TAG, SHA, TITLE, BODY)

    def test_literal_newline_is_rejected(self):
        with self.assertRaisesRegex(ValueError, '改行'):
            release.validate_input(REPO, TAG, SHA, TITLE, BODY + r'\n')

    def test_existing_markdown_contract(self):
        for name, title, body in (
            ('日本語のないタイトルを拒否する', 'Release', BODY),
            ('改行を含むタイトルを拒否する', TITLE + '\n続き', BODY),
            ('必須節の欠落を拒否する', TITLE, '説明だけ'),
            ('コメントだけの必須節を拒否する', TITLE, BODY.replace('内容を確認した。', '<!-- 説明 -->', 1)),
            ('必須節の重複を拒否する', TITLE, BODY + '\n## 検証\n確認した。'),
        ):
            with self.subTest(name=name), self.assertRaises(ValueError):
                release.validate_input(REPO, TAG, SHA, title, body)

    def test_tag_and_sha_boundaries(self):
        for tag in ('v0.0.1-alpha', '0.0.1', 'v1.01.0', 'v1.0.00', 'v1.0.1\n'):
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                release.validate_input(REPO, tag, SHA, TITLE, BODY)
        for target in ('main', 'g' * 40, SHA + '\n', 'a' * 64):
            with self.subTest(target=target), self.assertRaises(ValueError):
                release.validate_input(REPO, TAG, target, TITLE, BODY)
        release.validate_input(REPO, 'v10.2.0', SHA, TITLE, BODY)


class CreationTests(unittest.TestCase):
    def setUp(self):
        self.main_sha = SHA
        self.tags = []
        self.releases = []
        self.run = {'id': 7, 'head_sha': SHA, 'head_branch': 'main', 'event': 'push',
                    'status': 'completed', 'conclusion': 'success'}
        self.runs = [self.run]
        self.jobs = {'total_count': 1, 'jobs': [
            {'name': 'verify', 'status': 'completed', 'conclusion': 'success'}]}
        self.api = Mock(side_effect=self.read)
        self.send = Mock(return_value='https://github.com/example/project/releases/tag/v0.0.1\n')

    def read(self, endpoint, **kwargs):
        if endpoint.endswith('/git/ref/heads/main'):
            return {'object': {'sha': self.main_sha}}
        if '/git/matching-refs/tags/' in endpoint:
            return self.tags
        if '/releases?' in endpoint:
            return self.releases
        if '/actions/workflows/' in endpoint:
            return {'workflow_runs': self.runs}
        if '/actions/runs/7/jobs' in endpoint:
            return self.jobs
        raise AssertionError(f'予期しない読み取り: {endpoint}')

    def create(self, **kwargs):
        return release.create(REPO, TAG, SHA, TITLE, BODY,
                              api=self.api, send=self.send, **kwargs)

    def test_valid_create_reads_main_and_ci_then_sends_draft(self):
        self.assertEqual(self.send.return_value, self.create())
        self.api.assert_any_call(f'repos/{REPO}/git/ref/heads/main')
        self.api.assert_any_call(f'repos/{REPO}/actions/workflows/conventions.yml/runs?head_sha={SHA}&event=push&branch=main&per_page=1')
        self.api.assert_any_call(f'repos/{REPO}/actions/runs/7/jobs?per_page=100')
        self.send.assert_called_once_with(
            ['gh', 'release', 'create', TAG, f'--repo={REPO}', f'--target={SHA}',
             f'--title={TITLE}', '--notes-file=-', '--draft'], body=BODY)

    def test_non_main_target_does_not_write(self):
        self.main_sha = 'b' * 40
        with self.assertRaisesRegex(ValueError, 'main'):
            self.create()
        self.send.assert_not_called()

    def test_failed_ci_does_not_write(self):
        self.run['conclusion'] = 'failure'
        with self.assertRaisesRegex(ValueError, 'CI'):
            self.create()
        self.send.assert_not_called()

    def test_missing_ci_does_not_write(self):
        self.runs = []
        with self.assertRaisesRegex(ValueError, 'CI'):
            self.create()
        self.send.assert_not_called()

    def test_skipped_required_job_does_not_write(self):
        self.jobs['jobs'][0]['conclusion'] = 'skipped'
        with self.assertRaisesRegex(ValueError, 'verify'):
            self.create()
        self.send.assert_not_called()

    def test_incomplete_jobs_do_not_write(self):
        self.jobs['total_count'] = 101
        with self.assertRaisesRegex(ValueError, '不完全'):
            self.create()
        self.send.assert_not_called()

    def test_existing_tag_does_not_write(self):
        self.tags = [{'ref': f'refs/tags/{TAG}', 'object': {'type': 'commit', 'sha': SHA}}]
        with self.assertRaisesRegex(ValueError, '既存'):
            self.create()
        self.send.assert_not_called()

    def test_existing_release_does_not_write(self):
        self.releases = [{'tag_name': TAG, 'draft': True}]
        with self.assertRaisesRegex(ValueError, '既存'):
            self.create()
        self.send.assert_not_called()

    def test_changed_main_during_checks_does_not_write(self):
        def read(endpoint, **kwargs):
            data = self.read(endpoint, **kwargs)
            if '/jobs?' in endpoint:
                self.main_sha = 'b' * 40
            return data
        self.api.side_effect = read
        with self.assertRaisesRegex(ValueError, 'main'):
            self.create()
        self.send.assert_not_called()

    def test_check_only_never_writes(self):
        self.assertEqual('', self.create(check_only=True))
        self.api.assert_any_call(f'repos/{REPO}/actions/runs/7/jobs?per_page=100')
        self.send.assert_not_called()

    def test_invalid_ci_metadata_never_writes(self):
        for name, key, value in (
            ('別の SHA は拒否する', 'head_sha', 'b' * 40),
            ('PR の成功は代用しない', 'event', 'pull_request'),
            ('別ブランチは拒否する', 'head_branch', 'other'),
            ('実行中は拒否する', 'status', 'in_progress'),
            ('結論の欠落は拒否する', 'conclusion', None),
        ):
            with self.subTest(name=name):
                self.run[key] = value
                with self.assertRaises(ValueError):
                    self.create()
                self.run[key] = {'head_sha': SHA, 'event': 'push', 'head_branch': 'main',
                                 'status': 'completed', 'conclusion': 'success'}[key]
        self.send.assert_not_called()

    def test_missing_or_ambiguous_required_jobs_never_write(self):
        for name, jobs in (
            ('必須ジョブがない場合は拒否する', []),
            ('同名ジョブの重複を拒否する', self.jobs['jobs'] * 2),
            ('必須ジョブの実行中は拒否する', [{'name': 'verify', 'status': 'queued', 'conclusion': 'success'}]),
        ):
            with self.subTest(name=name):
                self.jobs = {'total_count': len(jobs), 'jobs': jobs}
                with self.assertRaises(ValueError):
                    self.create()
        self.send.assert_not_called()

    def test_api_failure_never_writes(self):
        self.api.side_effect = subprocess.TimeoutExpired(['gh', 'api'], 60)
        with self.assertRaises(subprocess.TimeoutExpired):
            self.create()
        self.send.assert_not_called()

    def test_input_failure_stops_before_api(self):
        with self.assertRaises(ValueError):
            release.create(REPO, TAG, SHA, TITLE, '本文なし', api=self.api, send=self.send)
        self.api.assert_not_called()
        self.send.assert_not_called()

    def test_prefix_matching_tag_is_not_a_duplicate(self):
        self.tags = [{'ref': f'refs/tags/{TAG}0'}]
        self.create()
        self.send.assert_called_once()


class PublicationTests(unittest.TestCase):
    read = CreationTests.read

    def setUp(self):
        CreationTests.setUp(self)
        self.draft = {'id': 9, 'tag_name': TAG, 'target_commitish': SHA, 'draft': True,
                      'prerelease': False, 'name': TITLE, 'body': BODY}
        self.releases = [self.draft]

    def publish(self, **kwargs):
        return release.publish(REPO, TAG, SHA, api=self.api, send=self.send, **kwargs)

    def test_valid_publish_confirms_published_release_and_tag(self):
        def send(args, **kwargs):
            self.draft['draft'] = False
            self.tags = [{'ref': f'refs/tags/{TAG}', 'object': {'type': 'commit', 'sha': SHA}}]
            return ''
        self.send.side_effect = send
        self.publish()
        self.send.assert_called_once_with(
            ['gh', 'release', 'edit', TAG, f'--repo={REPO}', '--draft=false'], body=None)

    def test_non_draft_cannot_be_published(self):
        self.draft['draft'] = False
        with self.assertRaisesRegex(ValueError, 'Draft'):
            self.publish()
        self.send.assert_not_called()

    def test_draft_with_different_target_cannot_be_published(self):
        self.draft['target_commitish'] = 'main'
        with self.assertRaisesRegex(ValueError, '対象'):
            self.publish()
        self.send.assert_not_called()

    def test_missing_draft_does_not_write(self):
        self.releases = []
        with self.assertRaisesRegex(ValueError, 'Release'):
            self.publish()
        self.send.assert_not_called()

    def test_wrong_existing_tag_stops_before_publication(self):
        self.tags = [{'ref': f'refs/tags/{TAG}', 'object': {'type': 'commit', 'sha': 'b' * 40}}]
        with self.assertRaisesRegex(ValueError, 'タグ'):
            self.publish()
        self.send.assert_not_called()

    def test_annotated_tag_is_resolved_to_commit(self):
        self.tags = [{'ref': f'refs/tags/{TAG}', 'object': {'type': 'tag', 'sha': 'b' * 40}}]
        original = self.api.side_effect
        def read(endpoint, **kwargs):
            if '/git/tags/' in endpoint:
                return {'object': {'type': 'commit', 'sha': SHA, 'url': 'https://api.github.com/object'}}
            return original(endpoint, **kwargs)
        self.api.side_effect = read
        self.send.side_effect = lambda *args, **kwargs: self.draft.update(draft=False) or ''
        self.publish()
        self.send.assert_called_once()

    def test_changed_draft_does_not_write(self):
        original = self.api.side_effect
        def read(endpoint, **kwargs):
            if '/jobs?' in endpoint:
                self.draft['body'] += '\n未検査の編集'
            return original(endpoint, **kwargs)
        self.api.side_effect = read
        with self.assertRaisesRegex(ValueError, '変わった'):
            self.publish()
        self.send.assert_not_called()

    def test_publish_check_only_never_writes(self):
        self.assertEqual('', self.publish(check_only=True))
        self.send.assert_not_called()

    def test_changed_notes_after_publication_are_reported(self):
        def send(*args, **kwargs):
            self.draft.update(draft=False, body='同時に変更された本文')
            self.tags = [{'ref': f'refs/tags/{TAG}', 'object': {'type': 'commit', 'sha': SHA}}]
            return ''
        self.send.side_effect = send
        with self.assertRaisesRegex(ValueError, '公開後'):
            self.publish()
        self.send.assert_called_once()

    def test_invalid_publish_target_stops_before_api(self):
        with self.assertRaises(ValueError):
            release.publish(REPO, TAG, 'main', api=self.api, send=self.send)
        self.api.assert_not_called()
        self.send.assert_not_called()

    def test_publication_preconditions_never_write(self):
        for name, change in (
            ('不正な保存済み本文を拒否する', lambda: self.draft.update(body='空欄')),
            ('プレリリースを拒否する', lambda: self.draft.update(prerelease=True)),
            ('重複する Release を拒否する', lambda: self.releases.append(copy.deepcopy(self.draft))),
            ('main の不一致を拒否する', lambda: setattr(self, 'main_sha', 'b' * 40)),
            ('CI の失敗を拒否する', lambda: self.run.update(conclusion='failure')),
        ):
            with self.subTest(name=name):
                self.setUp()
                change()
                with self.assertRaises(ValueError):
                    self.publish()
                self.send.assert_not_called()

    def test_tag_resolution_failures_never_write(self):
        for name, obj in (
            ('タグの循環を拒否する', {'type': 'tag', 'sha': 'b' * 40}),
            ('tree オブジェクトを拒否する', {'type': 'tree', 'sha': SHA}),
        ):
            with self.subTest(name=name):
                self.tags = [{'ref': f'refs/tags/{TAG}', 'object': obj}]
                original = self.read
                self.api.side_effect = lambda endpoint, **kwargs: (
                    {'object': obj} if '/git/tags/' in endpoint else original(endpoint, **kwargs))
                with self.assertRaises(ValueError):
                    self.publish()
        self.send.assert_not_called()

    def test_changed_main_before_publication_never_writes(self):
        def read(endpoint, **kwargs):
            data = self.read(endpoint, **kwargs)
            if '/jobs?' in endpoint:
                self.main_sha = 'b' * 40
            return data
        self.api.side_effect = read
        with self.assertRaises(ValueError):
            self.publish()
        self.send.assert_not_called()

    def test_write_failure_is_not_retried(self):
        self.send.side_effect = subprocess.TimeoutExpired(['gh', 'release', 'edit'], 60)
        with self.assertRaises(subprocess.TimeoutExpired):
            self.publish()
        self.send.assert_called_once()

    def test_missing_tag_after_publication_is_reported(self):
        self.send.side_effect = lambda *args, **kwargs: self.draft.update(draft=False) or ''
        with self.assertRaisesRegex(ValueError, 'タグ'):
            self.publish()
        self.send.assert_called_once()


if __name__ == '__main__':
    unittest.main()
