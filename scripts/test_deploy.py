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
        (202, 'application.yml', ['frontend-unit', 'go-lint', 'go-unit', 'go-integration', 'production-image']),
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

    def test_frontend_ci_must_succeed_before_image_publication(self):
        path = f'repos/{REPO}/actions/runs/202/jobs?per_page=100'
        for state in ['missing', 'failure', 'in_progress']:
            with self.subTest(state=state):
                data = ci_fixture()
                jobs = data[path]['jobs']
                frontend = next(job for job in jobs if job['name'] == 'frontend-unit')
                if state == 'missing':
                    jobs.remove(frontend)
                    data[path]['total_count'] = len(jobs)
                elif state == 'failure':
                    frontend['conclusion'] = 'failure'
                else:
                    frontend['status'] = 'in_progress'
                with self.assertRaises(ValueError):
                    deploy.preflight(REPO, TARGET, lambda key: copy.deepcopy(data[key]))

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


class PublishTests(unittest.TestCase):
    def exercise(self, fault=None):
        import json
        from pathlib import Path

        operations = []
        data = ci_fixture()
        main_reads = 0
        image_id = 'sha256:'+'1'*64
        digest = 'sha256:'+'2'*64

        def api(path):
            nonlocal main_reads
            if path.endswith('/git/ref/heads/main'):
                main_reads += 1
                if fault == 'main_changed' and main_reads > 1:
                    return {'object': {'sha': 'b'*40}}
            if fault == 'ci_failure' and '/jobs?' in path:
                raise OSError('API失敗')
            return copy.deepcopy(data[path])

        def run(args, **kwargs):
            operations.append(args)
            if args[:3] == ['git', 'rev-parse', 'HEAD']:
                return 'b'*40 if fault == 'head' else TARGET
            if args[:2] == ['git', 'status']:
                return ' M cmd/server/main.go' if fault == 'dirty' else ''
            if args[:2] == ['docker', 'build']:
                if fault == 'build':
                    raise OSError('ビルド失敗')
                Path(args[args.index('--iidfile')+1]).write_text('invalid' if fault == 'image_id' else image_id)
                return ''
            if args[:3] == ['docker', 'image', 'inspect']:
                return json.dumps({'Os': 'linux', 'Architecture': 'arm64' if fault == 'platform' else 'amd64'})
            if args[:2] == ['docker', 'tag']:
                self.assertEqual(args[2], image_id)
                return ''
            if args[:2] == ['docker', 'push']:
                if fault == 'push':
                    raise OSError('送信失敗')
                return '不明な出力' if fault == 'digest' else 'sha: digest: '+digest+' size: 1234'
            if args[:4] == ['docker', 'buildx', 'imagetools', 'inspect']:
                self.assertEqual(args[-1], f'ghcr.io/{REPO}@{digest}')
                if fault == 'manifest':
                    return '{}'
                return json.dumps({'config': {'digest': 'sha256:'+'3'*64 if fault == 'mismatch' else image_id}})
            self.fail(f'予期しないコマンド: {args}')

        def verify(image):
            operations.append(['verify', image])
            self.assertEqual(image, image_id)
            return 1 if fault == 'verify' else 0

        return operations, lambda: deploy.publish(REPO, TARGET, api=api, run=run, verify=verify)

    def test_build_once_verify_before_push_and_return_matching_digest(self):
        operations, publish = self.exercise()
        result = publish()
        self.assertEqual(result['image'], f'ghcr.io/{REPO}@sha256:'+ '2'*64)
        self.assertEqual(result['target'], TARGET)
        self.assertEqual(result['platform'], 'linux/amd64')
        self.assertEqual(sum(op[:2] == ['docker', 'build'] for op in operations), 1)
        build = next(op for op in operations if op[:2] == ['docker', 'build'])
        self.assertEqual(build[build.index('--platform')+1], 'linux/amd64')
        verify_index = next(i for i, op in enumerate(operations) if op[0] == 'verify')
        push_index = next(i for i, op in enumerate(operations) if op[:2] == ['docker', 'push'])
        self.assertLess(verify_index, push_index)

    def test_failure_before_push_does_not_write(self):
        for fault in ['ci_failure', 'head', 'dirty', 'build', 'image_id', 'platform', 'verify', 'main_changed']:
            with self.subTest(fault=fault):
                operations, publish = self.exercise(fault)
                with self.assertRaises((ValueError, OSError)):
                    publish()
                self.assertFalse(any(op[:2] == ['docker', 'push'] for op in operations))

    def test_write_or_postwrite_failure_is_not_retried(self):
        for fault in ['push', 'digest', 'manifest', 'mismatch']:
            with self.subTest(fault=fault):
                operations, publish = self.exercise(fault)
                with self.assertRaises((ValueError, OSError)):
                    publish()
                self.assertEqual(sum(op[:2] == ['docker', 'push'] for op in operations), 1)


class BlueprintTests(unittest.TestCase):
    image = f'ghcr.io/{REPO}@sha256:'+'2'*64

    def test_digest_free_single_service_and_explicit_public_url(self):
        data = deploy.blueprint(REPO, self.image, 'https://player-lens.onrender.com/tools', 'ghcr-reader')
        self.assertEqual(len(data['services']), 1)
        service = data['services'][0]
        self.assertEqual((service['runtime'], service['plan'], service['numInstances']), ('image', 'free', 1))
        self.assertEqual(service['healthCheckPath'], '/healthz')
        self.assertEqual(service['image'], {'url': self.image, 'creds': {'fromRegistryCreds': {'name': 'ghcr-reader'}}})
        env = {entry['key']: entry['value'] for entry in service['envVars']}
        self.assertEqual(env['MCP_PUBLIC_URL'], 'https://player-lens.onrender.com/tools')
        self.assertEqual(env['PORT'], '10000')
        self.assertEqual(env['MCP_MAX_CONCURRENT'], '2')
        self.assertNotIn('databases', data)
        self.assertNotIn('dockerfilePath', service)
        self.assertNotIn('disk', service)

    def test_bootstrap_remains_local_and_no_credential_is_invented(self):
        service = deploy.blueprint(REPO, self.image)['services'][0]
        env = {entry['key']: entry['value'] for entry in service['envVars']}
        self.assertEqual(env['MCP_PUBLIC_URL'], '')
        self.assertNotIn('creds', service['image'])

    def test_mutable_wrong_registry_and_wrong_repo_are_rejected(self):
        for image in ['ghcr.io/'+REPO+':latest', 'docker.io/'+REPO+'@sha256:'+'2'*64,
                      'ghcr.io/other/repo@sha256:'+'2'*64, self.image+'\n']:
            with self.subTest(image=image), self.assertRaises(ValueError):
                deploy.blueprint(REPO, image)

    def test_invalid_public_url_cannot_generate_broken_or_unsafe_config(self):
        for url in ['http://example.test/tools', 'https://example.test/healthz', 'https://u:p@example.test/tools',
                    'https://example.test/tools?', 'https://example.test/tools#', 'https://example.test/%74ools',
                    'https://example.test:0/tools', 'https://example.test:65536/tools', 'https://example.test:/tools',
                    'https://-bad.test/tools', 'https://example..test/tools', 'https://example.test/tools\n']:
            with self.subTest(url=url), self.assertRaises(ValueError):
                deploy.blueprint(REPO, self.image, url)


class CliTests(unittest.TestCase):
    def test_preflight_cli_only_reads_ci(self):
        import contextlib
        import io
        from unittest.mock import patch
        data=ci_fixture()
        with patch.object(deploy,'github_api',side_effect=lambda path:copy.deepcopy(data[path])), \
                patch.object(deploy,'publish') as publish, contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(deploy.main(['preflight','--repo',REPO,'--target',TARGET]),0)
            publish.assert_not_called()

    def test_publish_receipt_is_written_without_changing_returned_digest(self):
        import contextlib
        import io
        import json
        import tempfile
        from pathlib import Path
        from unittest.mock import patch
        result={'target':TARGET,'image_id':'sha256:'+'1'*64,'image':f'ghcr.io/{REPO}@sha256:'+'2'*64,'platform':'linux/amd64'}
        with tempfile.TemporaryDirectory() as directory:
            receipt=Path(directory)/'receipt.json'
            with patch.object(deploy,'publish',return_value=result), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(deploy.main(['publish','--repo',REPO,'--target',TARGET,'--receipt',str(receipt)]),0)
            self.assertEqual(json.loads(receipt.read_text()),result)
            with patch.object(deploy,'publish') as publish, contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(deploy.main(['publish','--repo',REPO,'--target',TARGET,'--receipt',str(receipt)]),1)
                publish.assert_not_called()

    def test_blueprint_from_receipt_is_created_without_network(self):
        import contextlib
        import io
        import json
        import tempfile
        from pathlib import Path
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as directory:
            receipt=Path(directory)/'receipt.json'
            receipt.write_text(json.dumps({'image':f'ghcr.io/{REPO}@sha256:'+'2'*64}))
            output=Path(directory)/'render.yaml'
            with patch.object(deploy,'github_api') as api, contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(deploy.main(['blueprint','--repo',REPO,'--receipt',str(receipt),'--output',str(output)]),0)
                api.assert_not_called()
            self.assertEqual(json.loads(output.read_text())['services'][0]['plan'],'free')

    def test_publish_error_is_not_retried_and_partial_receipt_is_removed(self):
        import contextlib
        import io
        import tempfile
        from pathlib import Path
        from unittest.mock import patch
        with tempfile.TemporaryDirectory() as directory:
            receipt=Path(directory)/'receipt.json'
            with patch.object(deploy,'publish',side_effect=ValueError('照合失敗')) as publish, contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(deploy.main(['publish','--repo',REPO,'--target',TARGET,'--receipt',str(receipt)]),1)
                publish.assert_called_once()
            self.assertFalse(receipt.exists())
