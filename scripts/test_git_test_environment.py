import contextlib
import io
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import test_staged_format
import test_validate
import verify_go_tools
import verify_hooks
import verify_secrets


class GitFixtureIsolationTests(unittest.TestCase):
    def test_fixture_operations_preserve_calling_repository_and_environment(self):
        cases = [
            ('規約検査の一時リポジトリ', test_validate.GitTests,
             'test_range_selects_new_commit'),
            ('整形検査の一時index', test_staged_format.StagedFormatTests,
             'test_checks_index_content_without_changing_worktree_or_index'),
            ('実Lefthook検証の一時リポジトリ', verify_hooks.HookTests,
             'test_pre_push_receives_stdin_and_rejects'),
            ('実Gitleaks検証の一時リポジトリ', verify_secrets.SecretToolTests,
             'test_clean_history_and_empty_or_deleted_push_are_accepted'),
            ('実Go整形検証の一時index', None, None),
        ]
        for name, fixture, method in cases:
            with self.subTest(name), tempfile.TemporaryDirectory() as directory:
                root = Path(directory)
                caller = root / 'caller'
                caller.mkdir()
                clean = {key: value for key, value in os.environ.items()
                         if not key.startswith('GIT_')}
                clean.update(GIT_CONFIG_NOSYSTEM='1', GIT_CONFIG_GLOBAL=os.devnull)

                def git(*args):
                    return subprocess.run(['git', *args], cwd=caller, env=clean,
                                          check=True, capture_output=True).stdout

                git('init', '-q', '--initial-branch=main')
                (caller / 'source.txt').write_text('保全する作業ファイル\n')
                git('add', 'source.txt')
                git('-c', 'user.name=呼び出し元', '-c', 'user.email=caller@example.invalid',
                    'commit', '-qm', 'caller baseline')

                def snapshot():
                    return (git('show-ref'),
                            *(p.read_bytes() for p in [caller / '.git/config',
                              caller / '.git/HEAD', caller / '.git/index', caller / 'source.txt']))

                before = snapshot()
                inherited = {**clean, 'GIT_DIR': str(caller / '.git'),
                             'GIT_COMMON_DIR': str(caller / '.git'),
                             'GIT_WORK_TREE': str(caller),
                             'GIT_INDEX_FILE': str(caller / '.git/index'),
                             'GIT_CONFIG_COUNT': '1', 'GIT_CONFIG_KEY_0': 'core.bare',
                             'GIT_CONFIG_VALUE_0': 'false'}
                result = unittest.TestResult()
                with patch.dict(os.environ, inherited, clear=True):
                    environment = dict(os.environ)
                    if fixture is not None:
                        fixture(method).run(result)
                    else:
                        def formatter(_, source):
                            if b';' in (source / 'source with spaces.go').read_bytes():
                                raise ValueError('未整形')
                        with patch.object(verify_go_tools.staged_format, 'format_snapshot', formatter), \
                                contextlib.redirect_stdout(io.StringIO()):
                            verify_go_tools.verify_staged_format(root)
                    self.assertEqual(environment, dict(os.environ), '呼び出し元の環境を保持する')
                self.assertEqual(before, snapshot(), '呼び出し元の設定・参照・index・ファイルを保持する')
                self.assertTrue(result.wasSuccessful(), result.errors + result.failures)


if __name__ == '__main__':
    unittest.main()
