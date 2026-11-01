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
from git_test_environment import isolated_git_environment


class GitFixtureIsolationTests(unittest.TestCase):
    def test_environment_is_restored_on_success_and_exception(self):
        for fails in (False, True):
            with self.subTest('例外時に復元する' if fails else '正常終了時に復元する'):
                inherited = {**os.environ, 'GIT_DIR': '/unused/caller',
                             'GIT_CONFIG_COUNT': '1', 'GIT_CONFIG_KEY_0': 'core.bare',
                             'GIT_CONFIG_VALUE_0': 'true', 'FIXTURE_INPUT': 'keep'}
                with patch.dict(os.environ, inherited, clear=True):
                    try:
                        with isolated_git_environment():
                            self.assertNotIn('GIT_DIR', os.environ)
                            self.assertNotIn('GIT_CONFIG_COUNT', os.environ)
                            self.assertEqual('keep', os.environ['FIXTURE_INPUT'])
                            if fails:
                                raise RuntimeError('検査の失敗')
                    except RuntimeError:
                        if not fails:
                            raise
                    self.assertEqual(inherited, dict(os.environ))

    def test_fixture_operations_preserve_calling_repository_and_environment(self):
        cases = [
            ('規約検査の一時リポジトリ', test_validate.GitTests,
             'test_range_selects_new_commit'),
            ('整形検査の一時index', test_staged_format.StagedFormatTests,
             'test_checks_index_content_without_changing_worktree_or_index'),
            ('実Lefthook検証の一時リポジトリ', verify_hooks.HookTests, None),
            ('実Gitleaks検証の一時リポジトリ', verify_secrets.SecretToolTests, None),
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
                global_config = root / 'global.gitconfig'
                global_config.write_text('[core]\n\tbare = true\n')
                inherited = {**clean, 'GIT_DIR': str(caller / '.git'),
                             'GIT_COMMON_DIR': str(caller / '.git'),
                             'GIT_WORK_TREE': str(caller),
                             'GIT_INDEX_FILE': str(caller / '.git/index'),
                             'GIT_CONFIG_COUNT': '1', 'GIT_CONFIG_KEY_0': 'core.bare',
                             'GIT_CONFIG_VALUE_0': 'false',
                             'GIT_CONFIG_GLOBAL': str(global_config)}
                result = unittest.TestResult()
                with patch.dict(os.environ, inherited, clear=True):
                    environment = dict(os.environ)
                    if fixture is not None and method is not None:
                        fixture(method).run(result)
                    elif fixture is not None:
                        command = verify_hooks.HookTests.command

                        def without_tool_install(instance, args, **kwargs):
                            if args == ['lefthook', 'install']:
                                return subprocess.CompletedProcess(args, 0, '', '')
                            return command(instance, args, **kwargs)

                        case = fixture()
                        with patch.object(verify_hooks.HookTests, 'command', without_tool_install):
                            try:
                                case.setUp()
                                observed = subprocess.check_output(
                                    ['git', 'rev-parse', '--show-toplevel'], cwd=case.repo,
                                    text=True).strip()
                                self.assertEqual(case.repo.resolve(), Path(observed).resolve())
                            finally:
                                case.doCleanups()
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
