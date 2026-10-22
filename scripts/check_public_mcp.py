"""公開 MCP の HTTP/JSON-RPC と明示的な少量収集を確認する。"""

import argparse
import json
import sys
import urllib.error
import urllib.request

from deploy import validate_public_url


def check(endpoint, *, request, app_id=None):
    """check は既定で実レビューを要求せず、明示時だけ1件・1ページを確認する。

    request は URL・HTTP method・JSON payload から status/headers/bytes を返す。
    再試行は行わず、結果へレビュー本文を含めない。
    """
    validate_public_url(endpoint)
    if app_id is not None and (type(app_id) is not int or not 1 <= app_id <= 4294967295):
        raise ValueError('App ID は1〜4294967295の整数で指定する')
    status, headers, body = request(endpoint[:-6]+'/healthz', 'GET', None)
    if status != 200 or body != b'ok\n' or not headers.get('Content-Type', '').startswith('text/plain'):
        raise ValueError('healthz の応答が異なる')

    def rpc(identifier, method, params):
        status, _, body = request(endpoint, 'POST', {'jsonrpc':'2.0','id':identifier,'method':method,'params':params})
        if status != 200:
            raise ValueError(f'MCP {method} は HTTP {status}。MCP_PUBLIC_URL と公開設定を確認する')
        response = json.loads(body)
        if (not isinstance(response, dict) or response.get('jsonrpc') != '2.0'
                or type(response.get('id')) is not int or response['id'] != identifier
                or 'error' in response or not isinstance(response.get('result'), dict)):
            raise ValueError('MCP の JSON-RPC 応答が不正')
        return response['result']

    try:
        initialized = rpc(1, 'initialize', {'protocolVersion':'2025-11-25','capabilities':{},
                                         'clientInfo':{'name':'player-lens-public-check','version':'1'}})
        if initialized.get('protocolVersion') != '2025-11-25':
            raise ValueError('MCP のプロトコル版が異なる')
        tools = rpc(2, 'tools/list', {}).get('tools')
        if not isinstance(tools, list) or [tool['name'] for tool in tools] != ['collect_reviews']:
            raise ValueError('MCP のツール一覧が異なる')
        invalid = rpc(3, 'tools/call', {'name':'collect_reviews','arguments':{'app_id':0}})
        if invalid.get('isError') is not True or invalid.get('structuredContent', {}).get('error_code') != 'invalid_input':
            raise ValueError('不正入力を取得前に拒否しなかった')
        identifiers = []
        if app_id is not None:
            result = rpc(4, 'tools/call', {'name':'collect_reviews','arguments':{
                'app_id':app_id,'language':'all','page_size':1,'max_reviews':1,'max_pages':1}})
            data = result['structuredContent']
            collection = data['collection']
            reviews = collection['reviews']
            if (result.get('isError') or data.get('error_code') != ''
                    or type(data['returned_count']) is not int or data['returned_count'] != 1
                    or type(collection['app_id']) is not int or collection['app_id'] != app_id
                    or type(collection['pages_fetched']) is not int or collection['pages_fetched'] != 1
                    or not isinstance(reviews, list) or len(reviews) != 1
                    or not isinstance(reviews[0]['id'], str) or not reviews[0]['id']):
                raise ValueError('明示した1件・1ページのレビュー取得を確認できなかった')
            identifiers = [reviews[0]['id']]
        return {'endpoint':endpoint,'protocol_version':'2025-11-25','steam_requested':app_id is not None,
                'returned_count':len(identifiers),'review_ids':identifiers}
    except (KeyError, TypeError, AttributeError) as error:
        raise ValueError('MCP の結果形式が不正') from error


class _NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


def request_url(url, method, payload):
    """request_url は認証なし・15秒・256 KiB上限で通信しリダイレクトしない。"""
    headers = {'Accept':'application/json, text/event-stream','MCP-Protocol-Version':'2025-11-25'}
    data = None
    if payload is not None:
        data = json.dumps(payload).encode('utf-8')
        headers['Content-Type'] = 'application/json'
    request = urllib.request.Request(url, data=data, headers=headers, method=method)
    opener = urllib.request.build_opener(_NoRedirect())
    try:
        response = opener.open(request, timeout=15)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        body = response.read(256*1024+1)
        if len(body) > 256*1024:
            raise ValueError('公開接続の応答量が上限を超えた')
        return response.status, dict(response.headers.items()), body


def main(argv=None):
    parser = argparse.ArgumentParser(description='公開 MCP を確認する。既定では実 Steam を呼ばない')
    parser.add_argument('endpoint', help='HTTPS の /tools URL')
    parser.add_argument('--collect-app-id', type=int, help='明示時だけ1件・1ページの実レビュー取得を確認する')
    args = parser.parse_args(argv)
    try:
        result = check(args.endpoint, request=request_url, app_id=args.collect_app_id)
    except (ValueError, OSError) as error:
        print(f'公開接続検査に失敗した: {error}', file=sys.stderr)
        return 1
    print(json.dumps(result, ensure_ascii=False))
    return 0


if __name__ == '__main__':
    sys.exit(main())
