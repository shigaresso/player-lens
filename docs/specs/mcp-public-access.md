# 認証なしの公開 MCP と許可 Host

公開 URL・ローカル／公開の許可条件と Host の照合は mcpaccess/domain、HTTP の保護は mcpaccess/adapter に置く。adapter.Protect は Access と次の handler を受け取る。 配置は [ADR-0013](../adr/0013-domain-folder-layers.md) に従う。

## 目的と状態

Issue #29 で、公開 MCP を OAuth なしで提供する。Go はレビュー収集を担当し、分析は接続元のクライアントが行う。LLM API キー・バックエンドの分析・利用コード・DB は導入しない。Issue #31 の作業中にユーザー配置の公開 HTTPS で匿名初期化・一覧・不正入力の拒否を確認した。検証済み digest の配置と実 ChatGPT の公開接続は未確認とする。実装と検証は [作業記録](../worklogs/29-public-mcp-access.md) に残す。

## 起動設定

`MCP_PUBLIC_URL` が空なら従来のローカル専用モードとする。指定時は認証なしの公開モードとし、HTTPS の正確な `/tools` URL を必須とする。資格情報・query（空も含む）・fragment（空も含む）・encoded path・空白・不正 authority を拒否する。ポートは1〜65535、DNS label と IP の正しい形式を要求する。不正設定は待ち受け前に非0終了し、診断に入力値を含めない。認証サービス・issuer・JWKS・JWT の設定は使用しない。

## HTTP の受付

- 公開設定時は設定した authority と localhost/loopback だけを受け付ける。DNS の大文字小文字、IP 表記、HTTPS の既定443を正規化して完全一致する。非既定ポートの違い・suffix・wildcard・転送ヘッダーによる迂回を許可しない。
- SDK の localhost 自動判定は公開設定時だけ無効化し、この Host 検査へ置き換える。両モードで Go の cross-origin 保護を維持する。不許可 Host・Origin は403で、収集と利用枠へ進めない。CORS は追加しない。
- Authorization ヘッダーを必要とせず、初期処理・ツール一覧・呼び出しを匿名で受け付ける。OAuth の metadata・challenge は公開せず、`/.well-known/oauth-protected-resource/tools` は404とする。
- `/tools` 以外のルートと `/healthz` は従来の契約を維持する。Host 検査は本人確認ではない。

### 保護の組み立ての境界（Issue #82）

`mcpaccess.New` は URL 未指定でも初期化済みのローカル用 Access を生成し、指定時は公開用 Access を生成する。未指定を nil で表す従来の Go API 契約を変更する。IsInitialized は生成済みか、IsPublic は公開用かを返す。nil とゼロ値は未初期化として扱い、ローカルの既定値として解釈しない。

両モードとも `mcpaccess/adapter.Protect(access, next)` が HTTP の保護を担当する。Access.AllowsHost は、ローカル用では従来の localHost、公開用では従来の authority 正規化と照合を使い、判定範囲を変えない。ProtectLocal は廃止する。`reviewmcp.NewHandler` は常に Protect を呼び、SDK の DisableLocalhostProtection は Access.IsPublic から決める。

Protect は有効な Access と非 nil の next を受け取り、未初期化の設定や nil の next は ErrInvalidConfig と nil handler を返す。Host の検査、Go の cross-origin 保護、next の順に処理する。Host の不許可では403と、ローカル用は「ローカルの Host のみ利用できます」、公開用は「許可されていない Host です」の説明文を返す。文言は組み立て時に選ぶ。Host または cross-origin で拒否した場合は next を呼ばず、許可時は next を1回だけ呼び、その応答を維持する。構築時に通信・時計・新しい資源を必要としない。

## 資源・共有範囲

[レビュー収集 MCP](review-mcp.md) と [利用制限](mcp-usage-limits.md) の件数・ページ・8秒・16 KiB本文・128 KiB応答・失敗・キャンセル・再試行なしを維持する。1つの handler の全クライアント・セッション・対応4版で同じ開始回数・同時枠（既定2件）を使う。1プロセスのメモリ内制限は再起動で消える。複数台の共有予算・本人確認・請求上限・拒否トラフィックの負荷対策を保証しない。

Web と共通の収集受付、初期同時数3の採否、DB による利用コード・残り回数、バックエンド LLM の費用制限は別 Issue とする。

## 検証

設定の境界・Host 正規化・転送ヘッダーと Origin の拒否はユニットテストで確認する。実 SDK の4版・複数セッションの匿名接続・一覧・呼び出し・共有枠を integration/race で確認する。同じ本番用イメージでローカルと公開設定の初期処理・一覧・不正入力、Host/Origin・metadata の404・不正設定・終了を確認する。実 Steam・実 LLM・外部認証サービスは自動テストから呼ばない。

生成済みのローカル用 Access と `Protect` の直接テストは MCP SDK や実ソケットを使わず、許可時の next の1回実行と応答維持、拒否時の非実行、Host の拒否理由と検査の優先順を確認する。`NewHandler` 経由の既存テストも維持し、保護の組み込みと SDK の本文上限・公開モードの契約を別途確認する。

[ADR-0009](../adr/0009-public-mcp-access.md) を参照する。

Issue #77 では Host/Origin の保護を Controller と区別し、middleware として mcpaccess/adapter に維持する。MCP の SDK 構築が保護と Controller を接続する。
