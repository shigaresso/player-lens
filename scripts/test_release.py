import unittest

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


if __name__ == '__main__':
    unittest.main()
