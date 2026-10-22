"""同じ検証済みイメージを使う配置準備を管理する。"""

import re


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
