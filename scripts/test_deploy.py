"""配置対象とイメージ公開の契約を外部通信なしで確認する。"""

import copy
import unittest

import deploy

REPO = 'shigaresso/player-lens'
TARGET = 'a' * 40


def ci_fixture():
    prefix = f'repos/{REPO}'
    data = {f'{prefix}/git/ref/heads/main': {'object': {'sha': TARGET}}}
    for identifier, workflow, names in [
        (101, 'conventions.yml', ['verify']),
        (202, 'application.yml', ['go-lint', 'go-unit', 'go-integration', 'production-image']),
    ]:
        data[f'{prefix}/actions/workflows/{workflow}/runs?head_sha={TARGET}&event=push&branch=main&per_page=1'] = {
            'workflow_runs': [{'id': identifier, 'head_sha': TARGET, 'head_branch': 'main',
                               'event': 'push', 'status': 'completed', 'conclusion': 'success'}]}
        data[f'{prefix}/actions/runs/{identifier}/jobs?per_page=100'] = {
            'total_count': len(names), 'jobs': [{'name': name, 'status': 'completed', 'conclusion': 'success'} for name in names]}
    return data


class PreflightTests(unittest.TestCase):
    def test_main_and_required_latest_ci_pass(self):
        data = ci_fixture()
        deploy.preflight(REPO, TARGET, lambda path: copy.deepcopy(data[path]))

    def test_invalid_reference_is_rejected_before_api(self):
        for repo, target in [('../repo', TARGET), (REPO, 'main'), (REPO, 'A'*40), (REPO, TARGET+'\n')]:
            with self.subTest(repo=repo, target=target):
                def api(path):
                    self.fail('不正入力で API を呼んだ')
                with self.assertRaises(ValueError):
                    deploy.preflight(repo, target, api)

    def test_mismatch_missing_failed_and_incomplete_ci_are_rejected(self):
        def change_main(data):
            data[f'repos/{REPO}/git/ref/heads/main']['object']['sha'] = 'b'*40

        mutations = [('別のmain', change_main)]
        for workflow in ['conventions.yml', 'application.yml']:
            path = f'repos/{REPO}/actions/workflows/{workflow}/runs?head_sha={TARGET}&event=push&branch=main&per_page=1'
            mutations.extend([
                (workflow+'の未検出', lambda data, path=path: data[path].update(workflow_runs=[])),
                (workflow+'の曖昧な取得', lambda data, path=path: data[path]['workflow_runs'].append(data[path]['workflow_runs'][0])),
            ])
            for field, value in [('head_sha', 'b'*40), ('head_branch', 'feature'), ('event', 'pull_request'),
                                 ('status', 'in_progress'), ('conclusion', 'failure')]:
                mutations.append((workflow+field, lambda data, path=path, field=field, value=value: data[path]['workflow_runs'][0].update({field:value})))
        for identifier in [101, 202]:
            path = f'repos/{REPO}/actions/runs/{identifier}/jobs?per_page=100'
            mutations.extend([
                (str(identifier)+'ジョブ不足', lambda data, path=path: data[path]['jobs'].pop()),
                (str(identifier)+'取得不完全', lambda data, path=path: data[path].update(total_count=101)),
                (str(identifier)+'ジョブ重複', lambda data, path=path: data[path].update(jobs=data[path]['jobs']*2, total_count=data[path]['total_count']*2)),
                (str(identifier)+'ジョブ失敗', lambda data, path=path: data[path]['jobs'][0].update(conclusion='failure')),
                (str(identifier)+'ジョブ未完了', lambda data, path=path: data[path]['jobs'][0].update(status='in_progress')),
            ])
        for name, mutate in mutations:
            with self.subTest(name=name):
                data = ci_fixture()
                mutate(data)
                with self.assertRaises(ValueError):
                    deploy.preflight(REPO, TARGET, lambda path: copy.deepcopy(data[path]))

    def test_api_error_is_not_success(self):
        def api(path):
            raise OSError('通信失敗')
        with self.assertRaises(OSError):
            deploy.preflight(REPO, TARGET, api)
