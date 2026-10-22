"""同じ検証済みイメージを使う配置準備を管理する。"""

import argparse
import ipaddress
import json
import re
import subprocess
import sys
import tempfile
from pathlib import Path
from urllib.parse import urlsplit

import check_image
from validate import github_api


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


def validate_public_url(value):
    """validate_public_url は資格情報のない HTTPS /tools URL を検査する。"""
    try:
        url = urlsplit(value)
        if (url.scheme != 'https' or not url.hostname or url.username is not None or url.password is not None
                or url.path != '/tools' or '?' in value or '#' in value
                or any(ord(char) <= 32 or ord(char) == 127 for char in value)
                or url.netloc.endswith(':') or url.port == 0):
            raise ValueError('公開 URL は資格情報のない HTTPS の /tools を指定する')
        host = url.hostname
        if '%' in host:
            raise ValueError('公開 Host が不正')
        try:
            ipaddress.ip_address(host)
        except ValueError:
            if len(host) > 253 or any(not re.fullmatch(r'[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?', label)
                                      for label in host.split('.')):
                raise ValueError('公開 Host が不正') from None
    except (ValueError, TypeError) as error:
        raise ValueError('公開 URL の形式が不正') from error


def blueprint(repo, image, public_url='', credential='', name='player-lens'):
    """blueprint は実 digest の Free 単一サービスを JSON 互換の YAML として構成する。"""
    if not re.fullmatch(r'[A-Za-z0-9][A-Za-z0-9_.-]*/[A-Za-z0-9][A-Za-z0-9_.-]*', repo):
        raise ValueError('リポジトリの形式が不正')
    if not re.fullmatch(re.escape(f'ghcr.io/{repo.lower()}@sha256:')+r'[0-9a-f]{64}', image):
        raise ValueError('このリポジトリの GHCR manifest digest が必要')
    if public_url:
        validate_public_url(public_url)
    if not re.fullmatch(r'[a-z][a-z0-9-]{0,62}', name):
        raise ValueError('サービス名の形式が不正')
    if len(credential) > 128 or any(ord(char) < 32 or ord(char) == 127 for char in credential):
        raise ValueError('registry 資格情報名の形式が不正')
    config = {'url': image}
    if credential:
        config['creds'] = {'fromRegistryCreds': {'name': credential}}
    values = {'PORT': '10000', 'MCP_PUBLIC_URL': public_url,
              'MCP_REQUESTS_PER_MINUTE': '6', 'MCP_REQUESTS_PER_DAY': '100', 'MCP_MAX_CONCURRENT': '2'}
    return {'services': [{'type': 'web', 'name': name, 'runtime': 'image', 'plan': 'free',
                          'region': 'oregon', 'numInstances': 1, 'healthCheckPath': '/healthz',
                          'image': config, 'envVars': [{'key': key, 'value': value} for key, value in values.items()]}]}


def command(args, timeout=60):
    """command は配置用の固定引数をリポジトリ内で実行する。"""
    return subprocess.run(args, cwd=Path(__file__).resolve().parents[1], check=True,
                          text=True, capture_output=True, timeout=timeout).stdout.strip()


def main(argv=None):
    parser = argparse.ArgumentParser(description='CI と検証済みイメージを使う Render 配置準備')
    commands = parser.add_subparsers(dest='mode', required=True)
    for mode in ['preflight', 'publish', 'blueprint']:
        child = commands.add_parser(mode)
        child.add_argument('--repo', required=True)
        if mode in ['preflight', 'publish']:
            child.add_argument('--target', required=True)
        if mode == 'publish':
            child.add_argument('--receipt', required=True)
        if mode == 'blueprint':
            source = child.add_mutually_exclusive_group(required=True)
            source.add_argument('--image')
            source.add_argument('--receipt')
            child.add_argument('--public-url', default='')
            child.add_argument('--credential', default='')
            child.add_argument('--name', default='player-lens')
            child.add_argument('--output', required=True)
    args = parser.parse_args(argv)
    created = None
    try:
        if args.mode == 'preflight':
            preflight(args.repo, args.target, github_api)
            result = {'target':args.target,'ci':'success'}
        elif args.mode == 'publish':
            output = Path(args.receipt)
            with output.open('x', encoding='utf-8') as stream:
                created = output
                result = publish(args.repo, args.target, api=github_api, run=command,
                                 verify=lambda image: check_image.main(['--image',image]))
                stream.write(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
        else:
            image = args.image
            if args.receipt:
                receipt = Path(args.receipt)
                if receipt.stat().st_size > 64*1024:
                    raise ValueError('receipt が大きすぎる')
                image = json.loads(receipt.read_text(encoding='utf-8'))['image']
            result = blueprint(args.repo, image, args.public_url, args.credential, args.name)
            output = Path(args.output)
            with output.open('x', encoding='utf-8') as stream:
                created = output
                stream.write(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as error:
        if created is not None:
            created.unlink(missing_ok=True)
        print(f'配置準備に失敗した: {error}', file=sys.stderr)
        if args.mode == 'publish':
            print('送信済みの可能性があるため registry の状態を確認する。自動再試行しない。',file=sys.stderr)
        return 1
    print(json.dumps(result, ensure_ascii=False))
    return 0


if __name__ == '__main__':
    sys.exit(main())
