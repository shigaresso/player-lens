"""Release の作成・公開を事前検査に接続する。"""

import argparse
import re
import subprocess
import sys
from pathlib import Path

import conventions
from github import send
from validate import github_api


SECTIONS = ('変更内容', '利用方法', '依存環境', '制約', '検証')


def validate_input(repo, tag, target, title, body):
    """validate_input は通信前に入力の形式を検査し、違反なら ValueError を返す。"""
    validate_reference(repo, tag, target)
    errors = conventions.validate_title(title)
    _, sections = conventions.validate_sections(body, SECTIONS)
    errors.extend(sections)
    if '\\n' in body:
        errors.append('本文の改行に文字列の \\n を含めない')
    if errors:
        raise ValueError('; '.join(errors))


def validate_reference(repo, tag, target):
    errors = []
    if not re.fullmatch(r'v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)', tag):
        errors.append('タグは先頭ゼロのない v数字.数字.数字 の正式版とする')
    if not re.fullmatch(r'[0-9a-f]{40}', target):
        errors.append('対象は40桁の完全な SHA を指定する')
    if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', repo):
        errors.append('リポジトリを OWNER/REPO の形式で指定する')
    if errors:
        raise ValueError('; '.join(errors))


def preflight(repo, target, api):
    """preflight は remote main と同じ SHA の規約 CI を読み取る。"""
    prefix = f'repos/{repo}'
    if api(f'{prefix}/git/ref/heads/main')['object']['sha'] != target:
        raise ValueError('対象 SHA が remote main の先端と一致しない')
    runs = api(f'{prefix}/actions/workflows/conventions.yml/runs?head_sha={target}&event=push&branch=main&per_page=1')['workflow_runs']
    if len(runs) != 1:
        raise ValueError('最新 CI が見つからない、または結果が不正')
    run = runs[0]
    expected = {'head_sha': target, 'head_branch': 'main', 'event': 'push',
                'status': 'completed', 'conclusion': 'success'}
    if any(run.get(key) != value for key, value in expected.items()):
        raise ValueError('同じ SHA の最新 push/main CI が成功していない')
    jobs = api(f"{prefix}/actions/runs/{run['id']}/jobs?per_page=100")
    if len(jobs['jobs']) != jobs['total_count']:
        raise ValueError('CI ジョブの取得が不完全（100件上限も確認する）')
    required = [job for job in jobs['jobs'] if job['name'] == 'verify']
    if len(required) != 1 or required[0].get('status') != 'completed' or required[0].get('conclusion') != 'success':
        raise ValueError('必須ジョブ verify が成功していない')


def create(repo, tag, target, title, body, *, api, send, check_only=False):
    """create は検査済みの同じ本文を渡し、正式版の Draft を作成する。"""
    validate_input(repo, tag, target, title, body)
    tags = api(f'repos/{repo}/git/matching-refs/tags/{tag}', paginate=True)
    if any(item['ref'] == f'refs/tags/{tag}' for item in tags):
        raise ValueError('既存タグを上書きしない')
    releases = api(f'repos/{repo}/releases?per_page=100', paginate=True)
    if any(item['tag_name'] == tag for item in releases):
        raise ValueError('既存 Release を上書きしない')
    preflight(repo, target, api)
    if api(f'repos/{repo}/git/ref/heads/main')['object']['sha'] != target:
        raise ValueError('検査中に main が変わったため最新の内容で再実行する')
    if check_only:
        return ''
    return send(['gh', 'release', 'create', tag, f'--repo={repo}', f'--target={target}',
                 f'--title={title}', '--notes-file=-', '--draft'], body=body)


def find_release(repo, tag, api):
    matches = [item for item in api(f'repos/{repo}/releases?per_page=100', paginate=True)
               if item['tag_name'] == tag]
    if len(matches) != 1:
        raise ValueError('対象 Release がない、または重複している')
    return matches[0].copy()


def verify_tag(repo, tag, target, api, *, required=False):
    matches = [item for item in api(f'repos/{repo}/git/matching-refs/tags/{tag}', paginate=True)
               if item['ref'] == f'refs/tags/{tag}']
    if not matches and not required:
        return
    if len(matches) != 1:
        raise ValueError('タグを一意に確認できない')
    obj = matches[0]['object']
    seen = set()
    while obj['type'] == 'tag':
        if obj['sha'] in seen or len(seen) >= 8:
            raise ValueError('タグの循環・深すぎる参照を解決できない')
        seen.add(obj['sha'])
        obj = api(f"repos/{repo}/git/tags/{obj['sha']}")['object']
    if obj['type'] != 'commit' or obj['sha'] != target:
        raise ValueError('タグが対象コミットを指していない、または確認できない')


def publish(repo, tag, target, *, api, send, check_only=False):
    """publish は Draft を読み取り、公開後の Release とタグを確認する。"""
    validate_reference(repo, tag, target)
    prefix = f'repos/{repo}'
    data = find_release(repo, tag, api)
    validate_input(repo, tag, target, data['name'], data['body'])
    if data['draft'] is not True or data['prerelease'] is not False:
        raise ValueError('正式版の Draft のみ公開できる')
    if data['target_commitish'] != target:
        raise ValueError('Draft の対象 SHA が明示した対象と一致しない')
    preflight(repo, target, api)
    verify_tag(repo, tag, target, api)
    if find_release(repo, tag, api) != data:
        raise ValueError('検査中に Draft が変わったため内容を確認して再実行する')
    if api(f'{prefix}/git/ref/heads/main')['object']['sha'] != target:
        raise ValueError('検査中に main が変わったため最新の内容で再実行する')
    if check_only:
        return ''
    result = send(['gh', 'release', 'edit', tag, f'--repo={repo}', '--draft=false'], body=None)
    published = find_release(repo, tag, api)
    verify_tag(repo, tag, target, api, required=True)
    if published['draft'] is not False or any(
        published.get(key) != data[key] for key in
        ('id', 'tag_name', 'target_commitish', 'name', 'body', 'prerelease')
    ):
        raise ValueError('公開後の Release またはタグの対象を確認できない')
    return result


def main(argv=None):
    parser = argparse.ArgumentParser(description='事前検査後に正式版の Draft Release を作成・公開する')
    commands = parser.add_subparsers(dest='mode', required=True)
    for mode in ('create', 'publish'):
        command = commands.add_parser(mode)
        command.add_argument('--repo', required=True)
        command.add_argument('--tag', required=True)
        command.add_argument('--target', required=True)
        command.add_argument('--check-only', action='store_true')
        if mode == 'create':
            command.add_argument('--title', required=True)
            command.add_argument('--body-file', required=True)
    args = parser.parse_args(argv)
    try:
        if args.mode == 'create':
            body = Path(args.body_file).read_text(encoding='utf-8')
            result = create(args.repo, args.tag, args.target, args.title, body,
                            api=github_api, send=send, check_only=args.check_only)
        else:
            result = publish(args.repo, args.tag, args.target,
                             api=github_api, send=send, check_only=args.check_only)
    except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as error:
        print(f'Release 検査・作成・公開のいずれかが失敗した: {error}', file=sys.stderr)
        print('自動再試行しない。書き込みや公開後確認の失敗では GitHub の状態を確認する。', file=sys.stderr)
        return 1
    print('Release 検査が成功した（書き込みなし）' if args.check_only else result.strip() or 'Release の確認が成功した')
    return 0


if __name__ == '__main__':
    sys.exit(main())
