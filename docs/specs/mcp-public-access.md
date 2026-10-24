# 認証なしの公開 MCP と許可 Host

## 目的と状態

Issue #29 で、公開 MCP を OAuth なしで提供する。Go はレビュー収集を担当し、分析は接続元のクライアントが行う。LLM API キー・バックエンドの分析・利用コード・DB は導入しない。Issue #31 の作業中にユーザー配置の公開 HTTPS で匿名初期化・一覧・不正入力の拒否を確認した。検証済み digest の配置と実 ChatGPT の公開接続は未確認とする。実装と検証は [作業記録](../worklogs/29-public-mcp-access.md) に残す。

## 起動設定

`MCP_PUBLIC_URL` が空なら従来のローカル専用モードとする。指定時は認証なしの公開モードとし、HTTPS の正確な `/tools` URL を必須とする。資格情報・query（空も含む）・fragment（空も含む）・encoded path・空白・不正 authority を拒否する。ポートは1〜65535、DNS label と IP の正しい形式を要求する。不正設定は待ち受け前に非0終了し、診断に入力値を含めない。認証サービス・issuer・JWKS・JWT の設定は使用しない。

## HTTP の受付

- 公開設定時は設定した authority と localhost/loopback だけを受け付ける。DNS の大文字小文字、IP 表記、HTTPS の既定443を正規化して完全一致する。非既定ポートの違い・suffix・wildcard・転送ヘッダーによる迂回を許可しない。
- SDK の localhost 自動判定は公開設定時だけ無効化し、この Host 検査へ置き換える。両モードで Go の cross-origin 保護を維持する。不許可 Host・Origin は403で、収集と利用枠へ進めない。CORS は追加しない。
- Authorization ヘッダーを必要とせず、初期処理・ツール一覧・呼び出しを匿名で受け付ける。OAuth の metadata・challenge は公開せず、`/.well-known/oauth-protected-resource/tools` は404とする。
- `/tools` 以外のルートと `/healthz` は従来の契約を維持する。Host 検査は本人確認ではない。

### 保護の組み立ての境界

ローカル専用モードは `mcpaccess.ProtectLocal(next)`、公開モードは `Access.Protect(next)` が HTTP の保護を担当する。`reviewmcp.NewHandler` は設定に応じてどちらかを選び、SDK の組み立てと共有する利用枠の所有を担当する。公開モードだけ SDK の localhost 保護を無効にする設定は維持する。

`ProtectLocal` の next は、呼び出し側が有効な非 nil の handler を渡す。従来の localhost/loopback の判定を使い、Host の検査、Go の cross-origin 保護、next の順に処理する。Host の不許可では403と「ローカルの Host のみ利用できます」の説明文を返す。Host または cross-origin で拒否した場合は next を呼ばず、許可時は next の応答を返す。構築時に通信・時計・新しい状態を必要とせず、公開側の authority 正規化へ判定を統一しない。

## 資源・共有範囲

[レビュー収集 MCP](review-mcp.md) と [利用制限](mcp-usage-limits.md) の件数・ページ・8秒・16 KiB本文・128 KiB応答・失敗・キャンセル・再試行なしを維持する。1つの handler の全クライアント・セッション・対応4版で同じ開始回数・同時枠（既定2件）を使う。1プロセスのメモリ内制限は再起動で消える。複数台の共有予算・本人確認・請求上限・拒否トラフィックの負荷対策を保証しない。

Web と共通の収集受付、初期同時数3の採否、DB による利用コード・残り回数、バックエンド LLM の費用制限は別 Issue とする。

## 検証

設定の境界・Host 正規化・転送ヘッダーと Origin の拒否はユニットテストで確認する。実 SDK の4版・複数セッションの匿名接続・一覧・呼び出し・共有枠を integration/race で確認する。同じ本番用イメージでローカルと公開設定の初期処理・一覧・不正入力、Host/Origin・metadata の404・不正設定・終了を確認する。実 Steam・実 LLM・外部認証サービスは自動テストから呼ばない。

`ProtectLocal` の直接テストは MCP SDK や実ソケットを使わず、許可時の next の1回実行と応答維持、拒否時の非実行、Host の拒否理由と検査の優先順を確認する。`NewHandler` 経由の既存テストも維持し、保護の組み込みと SDK の本文上限・公開モードの契約を別途確認する。

[ADR-0009](../adr/0009-public-mcp-access.md) を参照する。
