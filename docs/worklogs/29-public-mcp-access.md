# Issue #29：認証なしの公開 MCP

## 範囲と履歴

[Issue #29](https://github.com/shigaresso/player-lens/issues/29) の方針を OAuth なしへ更新した。旧 OAuth の9コミットは元ブランチに保護し、最新 origin/main `d0e974dffa0ae303fdb86e65a9b93bfd82283963` から `codex/29-public-mcp-access` を作成した。upstream は設定せず、push・PR 作成・Issue クローズは行わない。

今回は公開 URL と Host・Origin の制御、匿名 MCP と既存共有枠の維持だけを扱う。Web/DB/LLM・共通収集枠への移動・同時上限3への変更は含めない。

## 参照

文書一覧、公開デモ、公開 MCP、レビュー収集 MCP、HTTP サーバー、MCP 利用制限、ユースケース、ADR-0001・0007・0008・0009、開発手順を参照した。

## 検証記録

通常の Go コマンドは `docker compose run --rm --no-deps -T verify` 内で実行した。Red の記録は実際に失敗した理由を示す。

| 契約・コミット | Red | Green |
| --- | --- | --- |
| 公開 URL `6d3214e` | `go test -count=1 ./internal/mcpaccess` が実装ファイル未作成で失敗した | 同パッケージの全テストが成功した。ローカル既定、URL・Host・ポートと query/fragment の境界、診断への非公開を確認した |
| Host・Origin `81683bf` | `go test -count=1 ./internal/mcpaccess -run TestProtect` が Protect 未定義で失敗した | 同パッケージの全テストが成功した。匿名受付、完全一致・正規化・転送ヘッダー・Origin の拒否、nil/ゼロ値と challenge 非公開を確認した |
| サーバー接続 `3d04cdc` | `go test -count=1 ./cmd/server ./internal/reviewmcp -run 'TestReadMCPAccess\|TestPublicMCPAccess'` が NewHandler の公開設定引数未対応で失敗した | 3関連パッケージの全通常テストが成功した。公開設定の取得、匿名一覧・本文上限・Host/Origin、既存呼び出し元を確認した |
| SDK 回帰 `5151fa5` | 実装済みの契約を確認する回帰テストとして追加し、人工的な Red は作らなかった | `go test -tags=integration -race -count=1 ./internal/reviewmcp ./cmd/server -run 'TestPublicMCPProtocolsAndSharedBudget\|TestMCPRoute'` が成功した。4版・複数匿名セッション・共有枠・拒否の非消費・metadata 404 を確認した |

### 全体検証

`bash scripts/check-app.sh` が成功した。

- 整形・`go vet ./...`・全通常テストが成功した。
- 固定版 golangci-lint の通常対象と integration タグ付き対象が成功した（0 issues）。
- Git の失敗・不在・checkout 不在の3条件を確認し、全パッケージの `go test -tags=integration -race -count=1 ./...` が成功した。通常テストも含む。
- 一度ビルドした同じ本番 image ID `sha256:ad89d7e3dc99c5c8d4e482448e34249d1fe53591313f23717a37da739cb585e6` で、非 root・共有なし・read-only、ローカル2ポートと SIGTERM/SIGINT、公開設定の匿名初期化・一覧・不正入力・Host/Origin 拒否・metadata 404・SIGTERM を確認した。不正公開 URL の待ち受け前の非0終了と診断への非公開、PORT と既存3制限の不正設定の非0終了も成功した。
- 各コミットの必須 pre-commit / commit-msg を実行した。最終文書の末尾空行を pre-commit が検出したため修正して再実行した。Python 145件・静的検査・staged Go 整形・メッセージ検査が成功した。
- 差分、OAuth 用の直接依存・設定・HTTP challenge がないこと、変更文書のローカル参照先を AI がセルフレビューした。人によるレビューは未実施とする。

### 未実施と保証の範囲

Docker Desktop が停止していたため起動して検証した。実 Steam・LLM・外部認証サービスは呼ばなかった。Render 配置・公開 HTTPS・実 ChatGPT の公開接続は未確認とする。ソース共有なしのローカルイメージの成功を実配置の成功として扱わない。

Web と共通の収集枠・既定同時数3への変更・DB・利用コード・バックエンド LLM は未導入。既定の同時2件と1プロセスのメモリ内制限を維持した。再起動・複数台の共有予算と請求上限は保証しない。

フック・CI 自体は変更していないため、隔離した実フックの専用検証 `verify_hooks.py` は今回再実行していない。実コミットの必須フックは省略していない。push・pre-push・PR・実 GitHub アプリケーション CI は未実行とする。未追跡の `.DS_Store` は変更せずコミット対象から除外した。
