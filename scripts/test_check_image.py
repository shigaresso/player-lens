"""ビルド済みイメージを再ビルドせず検査する契約を確認する。"""

import contextlib
import io
import json
import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import MagicMock, patch

import check_image

IMAGE = 'sha256:' + 'a' * 64


class PromptBuilderImageTests(unittest.TestCase):
    def test_get_and_head_check_the_embedded_page(self):
        page = b'<form id="prompt-form"><input name="steam_url"><textarea id="prompt-output"></textarea></form><body data-input-schema="{}">function generatePrompt() {}</body>'
        with patch.object(check_image, 'request', side_effect=[
                (200, 'text/html; charset=utf-8', page),
                (200, 'text/html; charset=utf-8', b'')]) as request:
            check_image.check_prompt_builder(8080)
        self.assertEqual([('GET', '/'), ('HEAD', '/')],
                         [(call.args[1], call.args[2]) for call in request.call_args_list])

    def test_missing_page_wrong_type_and_head_body_are_rejected(self):
        good = (200, 'text/html; charset=utf-8', b'id="prompt-form" name="steam_url" id="prompt-output" data-input-schema="{}" function generatePrompt')
        for name, get, head in [
            ('画面がない', (404, 'text/plain', b''), (200, 'text/html; charset=utf-8', b'')),
            ('形式が違う', (200, 'application/json', good[2]), (200, 'text/html; charset=utf-8', b'')),
            ('フォームがない', (200, 'text/html; charset=utf-8', b''), (200, 'text/html; charset=utf-8', b'')),
            ('HEADに本文がある', good, (200, 'text/html; charset=utf-8', b'unexpected')),
        ]:
            with self.subTest(name=name), patch.object(check_image, 'request', side_effect=[get, head]):
                with self.assertRaises(RuntimeError):
                    check_image.check_prompt_builder(8080)


class MCPGuideTests(unittest.TestCase):
    def setUp(self):
        temporary = tempfile.TemporaryDirectory(prefix='player-lens-guide-test-')
        self.addCleanup(temporary.cleanup)
        root = Path(temporary.name)
        source = root / 'internal/reviewmcp/adapter/handler/protocol/analysis_guide.md'
        source.parent.mkdir(parents=True)
        self.instructions = '検査用の分析ガイド。根拠のレビュー ID を示してください。'
        source.write_text(self.instructions, encoding='utf-8')
        root_patch = patch.object(check_image, 'ROOT', root)
        root_patch.start()
        self.addCleanup(root_patch.stop)

    def guide(self):
        return {'version': '1', 'instructions': self.instructions}

    def check_result(self, output, *, text_output=None):
        if text_output is None:
            text_output = output
        results = [
            {'protocolVersion': '2025-11-25'},
            {'tools': [{'name': 'collect_reviews'}]},
            {'isError': True, 'structuredContent': output,
             'content': [{'type': 'text', 'text': json.dumps(text_output)}]},
        ]
        responses = [SimpleNamespace(status=200, read=lambda index=i, value=value:
                     json.dumps({'id': index, 'result': value}).encode())
                     for i, value in enumerate(results, 1)]
        responses.extend(SimpleNamespace(status=status, read=lambda: b'', getheader=lambda name: None)
                         for status in [403, 403, 404, 404])
        connection = MagicMock()
        connection.getresponse.side_effect = responses
        with patch.object(check_image.http.client, 'HTTPConnection', return_value=connection):
            check_image.check_mcp(8080)
        return connection

    def test_embedded_guide_is_checked_without_valid_steam_input(self):
        output = {'error_code': 'invalid_input', 'analysis_guide': self.guide()}
        connection = self.check_result(output)
        calls = connection.request.call_args_list
        tool_calls = [json.loads(call.args[2]) for call in calls
                      if len(call.args) > 2 and call.args[2] != '{}']
        self.assertEqual(tool_calls[-1]['params']['arguments'], {'app_id': 0})
        self.assertEqual(len(tool_calls), 3)
        self.assertEqual(connection.close.call_count, 7)

    def test_missing_changed_or_invalid_guide_is_rejected(self):
        for name, guide in [
            ('ガイドなし', None),
            ('版が異なる', {**self.guide(), 'version': '2'}),
            ('本文が異なる', {**self.guide(), 'instructions': '古いガイド'}),
            ('本文が文字列でない', {**self.guide(), 'instructions': 1}),
        ]:
            with self.subTest(name=name), self.assertRaises(RuntimeError):
                self.check_result({'error_code': 'invalid_input', 'analysis_guide': guide})

    def test_text_and_structured_guide_must_match(self):
        output = {'error_code': 'invalid_input', 'analysis_guide': self.guide()}
        with self.assertRaises(RuntimeError):
            self.check_result(output, text_output={'error_code': 'invalid_input'})


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


class PromptPageLogTests(unittest.TestCase):
    def records(self):
        return [dict(time='2026-11-02T00:00:00Z', level='INFO',
                     msg='プロンプト作成画面の応答を処理した',
                     **{'service.name': 'player-lens', 'event': 'prompt_page.completed',
                        'http.route': '/', 'http.request.method': method,
                        'http.response.status_code': 200, 'duration_ms': 0.25,
                        'outcome': 'success'}) for method in ['GET', 'HEAD']]

    def test_get_and_head_completion_logs_are_checked(self):
        output = 'SDK diagnostic\n' + '\n'.join(json.dumps(item) for item in
                                               [{'event': 'server.started'}, *self.records()])
        check_image.check_prompt_page_logs(output)

    def test_missing_duplicate_wrong_or_sensitive_logs_are_rejected(self):
        for name in ['missing', 'duplicate', 'status', 'route', 'duration', 'level', 'input']:
            with self.subTest(name=name):
                records = self.records()
                if name == 'missing':
                    records.pop()
                elif name == 'duplicate':
                    records.append(records[0])
                else:
                    key, value = {
                        'status': ('http.response.status_code', 500),
                        'route': ('http.route', '/?private=value'),
                        'duration': ('duration_ms', -1),
                        'level': ('level', 'ERROR'),
                        'input': ('query', 'private=value'),
                    }[name]
                    records[0][key] = value
                with self.assertRaises(RuntimeError):
                    check_image.check_prompt_page_logs('\n'.join(json.dumps(item) for item in records))
