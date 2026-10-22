"""公開 MCP のプローブを差し替えた応答だけで検証する。"""

import copy
import json
import unittest

import check_public_mcp

ENDPOINT = 'https://player-lens.onrender.com/tools'


def responses():
    return [
        (200, {'Content-Type': 'text/plain; charset=utf-8'}, b'ok\n'),
        (200, {}, json.dumps({'jsonrpc':'2.0','id':1,'result':{'protocolVersion':'2025-11-25'}}).encode()),
        (200, {}, json.dumps({'jsonrpc':'2.0','id':2,'result':{'tools':[{'name':'collect_reviews'}]}}).encode()),
        (200, {}, json.dumps({'jsonrpc':'2.0','id':3,'result':{'isError':True,'structuredContent':{'error_code':'invalid_input'}}}).encode()),
    ]


def requester(queue, calls):
    def request(url, method, payload):
        calls.append((url,method,payload))
        return queue.pop(0)
    return request


class ProtocolTests(unittest.TestCase):
    def test_default_probes_protocol_without_valid_steam_request(self):
        calls=[]
        result=check_public_mcp.check(ENDPOINT,request=requester(responses(),calls))
        self.assertEqual(len(calls),4)
        self.assertEqual(calls[0][0],'https://player-lens.onrender.com/healthz')
        self.assertEqual(calls[-1][2]['params']['arguments'],{'app_id':0})
        self.assertFalse(result['steam_requested'])
        self.assertEqual(result['review_ids'],[])

    def test_explicit_collect_is_one_review_and_page_and_does_not_output_text(self):
        queue=responses()
        queue.append((200,{},json.dumps({'jsonrpc':'2.0','id':4,'result':{'structuredContent':{
            'error_code':'','returned_count':1,'collection':{'app_id':4800590,'pages_fetched':1,
            'reviews':[{'id':'review-1','text':'外部本文を出力へ転載しない'}]}}}}).encode()))
        calls=[]
        result=check_public_mcp.check(ENDPOINT,request=requester(queue,calls),app_id=4800590)
        args=calls[-1][2]['params']['arguments']
        self.assertEqual((args['app_id'],args['page_size'],args['max_reviews'],args['max_pages']),(4800590,1,1,1))
        self.assertEqual(result['review_ids'],['review-1'])
        self.assertNotIn('外部本文',str(result))

    def test_bad_endpoint_and_app_id_are_rejected_before_http(self):
        for endpoint, app_id in [('http://example.test/tools',None),(ENDPOINT,0),(ENDPOINT,True),(ENDPOINT,4294967296)]:
            with self.subTest(endpoint=endpoint, app_id=app_id):
                calls=[]
                with self.assertRaises(ValueError):
                    check_public_mcp.check(endpoint,request=requester([],calls),app_id=app_id)
                self.assertEqual(calls,[])

    def test_http_protocol_and_result_failures_are_not_success(self):
        cases=[('healthz失敗',0,(503,{},b'busy')),
               ('healthz本文不正',0,(200,{},b'not ok')),
               ('ローカル専用',1,(403,{},b'local only')),
               ('リダイレクト',1,(302,{'Location':'https://elsewhere.test'},b'')),
               ('JSON不正',1,(200,{},b'bad')),
               ('別ID',1,(200,{},b'{"jsonrpc":"2.0","id":2,"result":{}}')),
               ('RPCエラー',1,(200,{},b'{"jsonrpc":"2.0","id":1,"error":{}}')),
               ('別プロトコル',1,(200,{},b'{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"other"}}')),
               ('ツール不足',2,(200,{},b'{"jsonrpc":"2.0","id":2,"result":{"tools":[]}}')),
               ('入力拒否なし',3,(200,{},b'{"jsonrpc":"2.0","id":3,"result":{}}'))]
        for name,index,response in cases:
            with self.subTest(name=name):
                queue=copy.deepcopy(responses());queue[index]=response
                calls=[]
                with self.assertRaises(ValueError):
                    check_public_mcp.check(ENDPOINT,request=requester(queue,calls))
                self.assertEqual(len(calls),index+1)

    def test_network_error_propagates_without_retry(self):
        calls=[]
        def request(*args):
            calls.append(args)
            raise OSError('通信失敗')
        with self.assertRaises(OSError):
            check_public_mcp.check(ENDPOINT,request=request)
        self.assertEqual(len(calls),1)


class TransportTests(unittest.TestCase):
    def test_request_uses_timeout_bounds_response_and_closes_it(self):
        from unittest.mock import MagicMock, patch
        response=MagicMock()
        response.__enter__.return_value=response
        response.status=200
        response.headers={'Content-Type':'application/json'}
        response.read.return_value=b'{}'
        with patch('urllib.request.build_opener') as factory:
            factory.return_value.open.return_value=response
            status,_,body=check_public_mcp.request_url(ENDPOINT,'POST',{'id':1})
            args,kwargs=factory.return_value.open.call_args
            self.assertEqual(kwargs['timeout'],15)
            self.assertEqual(args[0].method,'POST')
            self.assertIsNone(args[0].get_header('Authorization'))
            self.assertEqual(json.loads(args[0].data),{'id':1})
            response.read.assert_called_once_with(256*1024+1)
            response.__exit__.assert_called_once()
            self.assertIsNone(factory.call_args.args[0].redirect_request(None,None,302,'',{},'https://elsewhere.test'))
        self.assertEqual((status,body),(200,b'{}'))

    def test_oversized_response_is_rejected_and_closed(self):
        from unittest.mock import MagicMock, patch
        response=MagicMock()
        response.__enter__.return_value=response
        response.read.return_value=b'x'*(256*1024+1)
        with patch('urllib.request.build_opener') as factory:
            factory.return_value.open.return_value=response
            with self.assertRaises(ValueError):
                check_public_mcp.request_url(ENDPOINT,'GET',None)
            response.__exit__.assert_called_once()

    def test_http_error_is_returned_for_protocol_check_and_closed(self):
        import io
        from urllib.error import HTTPError
        from unittest.mock import patch
        body=io.BytesIO(b'local only')
        error=HTTPError(ENDPOINT,403,'Forbidden',{},body)
        with patch('urllib.request.build_opener') as factory:
            factory.return_value.open.side_effect=error
            result=check_public_mcp.request_url(ENDPOINT,'GET',None)
        self.assertEqual(result[0],403)
        self.assertTrue(body.closed)

    def test_cli_reports_failure_without_retry(self):
        import contextlib
        import io
        from unittest.mock import patch
        with patch.object(check_public_mcp,'check',side_effect=ValueError('HTTP 403')) as check, \
                contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(check_public_mcp.main([ENDPOINT]),1)
            check.assert_called_once()


class HeaderTests(unittest.TestCase):
    def test_header_names_are_case_insensitive(self):
        queue=responses()
        queue[0]=(200,{'content-type':'text/plain; charset=utf-8'},b'ok\n')
        self.assertFalse(check_public_mcp.check(ENDPOINT,request=requester(queue,[]))['steam_requested'])


class CollectionFailureTests(unittest.TestCase):
    def test_empty_limited_or_wrong_collection_is_not_reported_as_success(self):
        base = {'structuredContent': {'error_code': '', 'returned_count': 1,
                'collection': {'app_id': 4800590, 'pages_fetched': 1, 'reviews': [{'id': 'review-1'}]}}}
        cases = [
            ('利用枠の拒否', lambda result: result.update(isError=True)),
            ('空の結果', lambda result: result['structuredContent'].update(returned_count=0)),
            ('別の作品', lambda result: result['structuredContent']['collection'].update(app_id=440)),
            ('ページ数超過', lambda result: result['structuredContent']['collection'].update(pages_fetched=2)),
            ('件数超過', lambda result: result['structuredContent']['collection']['reviews'].append({'id': 'review-2'})),
            ('IDなし', lambda result: result['structuredContent']['collection']['reviews'][0].update(id='')),
        ]
        for name, mutate in cases:
            with self.subTest(name=name):
                result = copy.deepcopy(base)
                mutate(result)
                queue = responses()
                queue.append((200, {}, json.dumps({'jsonrpc': '2.0', 'id': 4, 'result': result}).encode()))
                calls = []
                with self.assertRaises(ValueError):
                    check_public_mcp.check(ENDPOINT, request=requester(queue, calls), app_id=4800590)
                self.assertEqual(len(calls), 5)
