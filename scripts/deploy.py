"""同じ検証済みイメージを使う配置準備を管理する。"""

import json
import re
import tempfile
from pathlib import Path


def validate_target(repo, target):
    """validate_target は通信前にリポジトリと完全な SHA を検査する。"""
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*', repo):
        raise ValueError('リポジトリは OWNER/REPO で指定する')
    if not re.fullmatch(r'[0-9a-f]{40}', target):
        raise ValueError('対象は40桁の完全な SHA で指定する')


def preflight(repo, target, api):
    """preflight は main と同じ SHA の最新 push CI と必要ジョブを検査する。"""
    validate_target(repo, target)
    prefix = f'repos/{repo}'
    try:
        if api(f'{prefix}/git/ref/heads/main')['object']['sha'] != target:
            raise ValueError('対象が remote main の先端と一致しない')
        workflows = {
            'conventions.yml': {'verify'},
            'application.yml': {'go-lint', 'go-unit', 'go-integration', 'production-image'},
        }
        for workflow, required in workflows.items():
            runs = api(f'{prefix}/actions/workflows/{workflow}/runs?head_sha={target}&event=push&branch=main&per_page=1')['workflow_runs']
            if not isinstance(runs, list) or len(runs) != 1:
                raise ValueError('最新 CI が見つからないか応答が曖昧')
            run = runs[0]
            expected = {'head_sha': target, 'head_branch': 'main', 'event': 'push',
                        'status': 'completed', 'conclusion': 'success'}
            if any(run.get(key) != value for key, value in expected.items()):
                raise ValueError('同じ SHA の最新 push/main CI が成功していない')
            if type(run['id']) is not int or run['id'] < 1:
                raise ValueError('CI の識別子が不正')
            jobs = api(f"{prefix}/actions/runs/{run['id']}/jobs?per_page=100")
            if not isinstance(jobs['jobs'], list) or type(jobs['total_count']) is not int or len(jobs['jobs']) != jobs['total_count']:
                raise ValueError('CI ジョブの取得が不完全')
            for name in required:
                matches = [job for job in jobs['jobs'] if job['name'] == name]
                if len(matches) != 1 or matches[0].get('status') != 'completed' or matches[0].get('conclusion') != 'success':
                    raise ValueError('必要な CI ジョブが一意に成功していない')
    except (KeyError, TypeError, AttributeError) as error:
        raise ValueError('CI の応答形式が不正') from error


def publish(repo, target, *, api, run, verify):
    """publish は一度ビルドしたイメージだけを検証・送信し digest を照合する。

    run は引数リストを実行して stdout を返す。verify は同じ image ID の
    本番検査を行い、成功時に0を返す。書き込みを自動再試行しない。
    """
    preflight(repo, target, api)
    if run(['git', 'rev-parse', 'HEAD']).strip() != target:
        raise ValueError('checkout と対象 SHA が一致しない')
    if run(['git', 'status', '--porcelain', '--untracked-files=all']).strip():
        raise ValueError('イメージ公開には clean な checkout が必要')
    with tempfile.TemporaryDirectory(prefix='player-lens-publish-') as directory:
        iid = Path(directory) / 'image-id'
        run(['docker', 'build', '--platform', 'linux/amd64', '--target', 'production',
             '--label', f'org.opencontainers.image.source=https://github.com/{repo}',
             '--label', f'org.opencontainers.image.revision={target}',
             '--iidfile', str(iid), '.'], timeout=900)
        image_id = iid.read_text(encoding='utf-8').strip()
        if not re.fullmatch(r'sha256:[0-9a-f]{64}', image_id):
            raise ValueError('ビルド結果の image ID が不正')
        config = json.loads(run(['docker', 'image', 'inspect', '--format', '{{json .}}', image_id]))
        if config.get('Os') != 'linux' or config.get('Architecture') != 'amd64':
            raise ValueError('Render の実行対象は linux/amd64 とする')
        if verify(image_id) != 0:
            raise ValueError('本番イメージの検証が失敗したため公開しない')
        if api(f'repos/{repo}/git/ref/heads/main')['object']['sha'] != target:
            raise ValueError('検証中に main が変わったため公開しない')
        image_name = f'ghcr.io/{repo.lower()}'
        tag = f'{image_name}:sha-{target}-{image_id[7:]}'
        run(['docker', 'tag', image_id, tag])
        output = run(['docker', 'push', tag], timeout=600)
        digests = re.findall(r'\bdigest:\s+(sha256:[0-9a-f]{64})(?:\s|$)', output)
        if len(digests) != 1:
            raise ValueError('送信済みの可能性があるため digest を確認し自動再試行しない')
        image = f'{image_name}@{digests[0]}'
        manifest = json.loads(run(['docker', 'buildx', 'imagetools', 'inspect', '--raw', image]))
        config = manifest.get('config') if isinstance(manifest, dict) else None
        if not isinstance(config, dict) or config.get('digest') != image_id:
            raise ValueError('送信した manifest と検証済み image ID が一致しない')
        return {'target': target, 'image_id': image_id, 'image': image, 'platform': 'linux/amd64'}
