"""ビルド済みイメージを再ビルドせず検査する契約を確認する。"""

import contextlib
import io
import unittest
from unittest.mock import patch

import check_image

IMAGE = 'sha256:' + 'a' * 64


class ExistingImageTests(unittest.TestCase):
    def test_existing_image_is_checked_without_building(self):
        @contextlib.contextmanager
        def container(*args):
            yield 'container'

        def docker(*args, **kwargs):
            if args[:2] == ('image', 'inspect'):
                return '{"User":"65532:65532"}'
            if args[0] == 'wait':
                return '1'
            if args[0] == 'logs':
                return 'MCP_PUBLIC_URL PORT MCP_REQUESTS_PER_MINUTE MCP_REQUESTS_PER_DAY MCP_MAX_CONCURRENT'
            self.fail(f'予期しない Docker 操作: {args}')

        with patch.object(check_image, 'docker', side_effect=docker) as command, \
                patch.object(check_image, 'container', side_effect=container), \
                patch.object(check_image, 'check_running') as running, \
                contextlib.redirect_stdout(io.StringIO()):
            self.assertEqual(check_image.main(['--image', IMAGE]), 0)
        self.assertEqual(running.call_count, 3)
        self.assertTrue(all(call.args[0] == IMAGE for call in running.call_args_list))
        self.assertFalse(any(call.args[0] == 'build' for call in command.call_args_list))

    def test_mutable_or_invalid_image_reference_is_rejected_before_docker(self):
        for image in ['latest', 'player-lens:main', 'sha256:short', IMAGE+'\n', 'sha256:'+'A'*64]:
            with self.subTest(image=image), patch.object(check_image, 'docker') as command, \
                    contextlib.redirect_stderr(io.StringIO()):
                self.assertEqual(check_image.main(['--image', image]), 1)
                command.assert_not_called()

    def test_existing_image_verification_failure_is_nonzero(self):
        with patch.object(check_image, 'docker', return_value='{"User":"65532:65532"}'), \
                patch.object(check_image, 'check_running', side_effect=RuntimeError('失敗')), \
                contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(check_image.main(['--image', IMAGE]), 1)
