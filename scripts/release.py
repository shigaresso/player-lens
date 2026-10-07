"""Release の作成・公開を事前検査に接続する。"""

import re

import conventions


SECTIONS = ('変更内容', '利用方法', '依存環境', '制約', '検証')


def validate_input(repo, tag, target, title, body):
    """validate_input は通信前に入力の形式を検査し、違反なら ValueError を返す。"""
    errors = conventions.validate_title(title)
    _, sections = conventions.validate_sections(body, SECTIONS)
    errors.extend(sections)
    if not re.fullmatch(r'v(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)', tag):
        errors.append('タグは先頭ゼロのない v数字.数字.数字 の正式版とする')
    if not re.fullmatch(r'[0-9a-f]{40}', target):
        errors.append('対象は40桁の完全な SHA を指定する')
    if not re.fullmatch(r'[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+', repo):
        errors.append('リポジトリを OWNER/REPO の形式で指定する')
    if '\\n' in body:
        errors.append('本文の改行に文字列の \\n を含めない')
    if errors:
        raise ValueError('; '.join(errors))
