# Controller を handler 直下で見つけられる配置に整理した

## 背景・目的

[Issue #77](https://github.com/shigaresso/player-lens/issues/77) の目的を、handler 直下から Controller と処理全体の流れを見つけられることへ訂正した。先行したローカル実装では生成・middleware・補助処理まで同列に移していたため、既存の2コミットを保持して訂正コミットを追加した。

前提の PR #78 が統合された origin/main（264b96c7e0621341175a2a74b914514b0a9a6ba2）から作成した codex/77-route-handler-layout で作業した。upstream は未設定で、push・PR 作成・Issue のクローズは行っていない。旧配置の検証記録を今回の最終構成の保証として流用せず、訂正後の構成で全体検証を実行した。

## 変更内容

| 場所 | 担当する処理 |
| --- | --- |
| cmd/server/adapter/handler/health.go | 生存確認の応答・計測・結果記録をつなぐ Controller |
| 同 handler/logging | 完了イベントの属性・level・安全な原因分類 |
| internal/reviewmcp/adapter/handler/collect_reviews.go | context の準備・入力変換・UseCase・応答変換をつなぐ Controller |
| 同 handler/protocol | 入力・応答・公開失敗・schema・分析ガイドの詳細 |
| internal/reviewmcp/adapter/server.go・collection_definition.go | SDK・共有 limiter の生成、Server の設定・登録、ツール定義 |
| internal/mcpaccess/adapter | Host/Origin を検査して next へ渡す middleware |

handler 直下の本番 Go ファイルは health.go と collect_reviews.go の各1つになった。MCP の context 連携と collect を同じファイルへまとめ、入口を委譲するだけのラッパーにしていない。専用の詳細をサブフォルダへ移し、VS Code のファイルネスティングも Controller と関連テストをまとめる設定に更新した。

protocol は ParseInput・InvalidInputResult・EncodeResult・InputSchema・OutputSchema と、応答量超過を識別する ErrResponseTooLarge を接続点として公開した。受信型・出力型・分類の詳細は非公開に維持し、親の handler・adapter を参照しない。構築側も同じ schema を利用するため、上限やガイドを重複定義していない。

healthz は Controller が固定の HealthPath と結果を logging.HealthCheck へ渡す。登録とログのパスの定義を共有し、logging から親を参照しない。テスト間で共有していた fixture は adapter/internal/testutil に移し、既存の依存検査で本番コードからの参照を禁止した。循環依存、空のパッケージ、互換ラッパー、将来用の依存は追加していない。

[受付とルーティングの一覧](../handler-routing.md)、ADR-0013・0017・0009、関連仕様と操作一覧を最終構成へ更新した。外側の RegisterRoutes、domain/application、CLI、外部クライアント、JSON 出力の責務を維持した。

## 検証

### 配置規則の Red・Green

配置規則の入力例を先に訂正し、従来の検査では生成・middleware の外側配置など5例の期待結果と一致しない Red を確認した。検査を Controller の実装と直下の一覧性へ訂正すると、旧配置の本番12ファイルが違反として検出された。配置の訂正後に Green を確認した。16入力例で Controller の外側配置・詳細の直下配置・奥に隠れた Controller を拒否し、外側の生成・middleware と詳細のサブフォルダを許可した。

検査は構文を使い、SDK 内部や任意の動的な型の全ディスパッチを証明しない。既存の TestLayerDependencies と併用し、循環を含む不正な依存は Go のビルドと合わせて確認した。

関連テストの実行で、移動前の NewHandler を参照していたテストの配置漏れを修正した。最初の全体検証では lint が応答変換関数の空行1箇所を指摘したため修正した。既存の入力・期待値、リンター設定、検査対象を緩めて通すことはしていない。

### 最終構成の全体検証

- Docker Compose の verify で関連パッケージの通常テストが成功した。
- `bash scripts/check-app.sh` が成功した。gofmt・vet・全通常テスト、lint（0 issues）、Git 非依存3条件、race 付き通常／結合テストを実行した。結合テストだけの結果とは扱わない。
- 同じ本番用 image ID `sha256:a2650bd5626a8d5a805cfbd0a4412774e3359aa0a16c7b10dd4153d3c17f3a12` で HTTP/MCP、Host/Origin、PORT・公開 URL・利用上限の不正設定、SIGTERM/SIGINT を確認した。開発用のソース共有を使わず、移動後の分析ガイドの埋め込みも確認した。
- 訂正前のコミット682aacbと比較し、既存の Test 関数151個の名前と個数を保持していることを確認した。分析ガイドの本文は基点 origin/main とバイト単位で一致し、版も維持した。
- 配置訂正と検証記録のコミットで必須フックを実行した。Python177テスト・静的検査・必要な index の gofmt・commit-msg、最終の差分・コミット範囲・本文・ローカルリンクの検査結果は Issue に記録した。

### AI のセルフレビューと制約

入力・schema・応答変換・context 連携の差分と登録元を確認した。共有 limiter の生成位置と期間、応答変換まで同時枠を保持する同期 callback、HTTP/SDK のキャンセル連携と callback 解放、失敗原因の保持と公開結果を引き継いだ。HTTP の listener・接続・終了待機、取得クライアントの解放担当を維持した。ログを検証しないテストの破棄出力と、検証するテストの捕捉出力も引き継いだ。人によるレビューは未実施。

Docker と GitHub 認証を権限付きで実行し、検証用の一時 Docker 設定を使用した。必須フックと検査を省略していない。push 前のため pre-push・コード CI・送信済み head/base を用いた PR 作成検査は未実施。実 Render 配置・外部サービス接続、coverage 閾値・分岐・mutation（Stryker を含む）は導入・実行していない。

## 参照仕様・ADR

- [受付とルーティング](../handler-routing.md) と [ADR-0013](../adr/0013-domain-folder-layers.md) の Controller・詳細・構築・middleware の境界
- [HTTP サーバー仕様](../specs/http-server.md) の応答・構造化ログ・生成と登録・終了・検証、および [ADR-0017](../adr/0017-health-structured-logging.md)
- [レビュー収集 MCP](../specs/review-mcp.md) の入力・ツール登録と実行・結果・失敗・資源、[ユースケース](../use-cases.md) の操作境界
- [公開 MCP 仕様](../specs/mcp-public-access.md) と [ADR-0009](../adr/0009-public-mcp-access.md) の受付保護、[ADR-0008](../adr/0008-mcp-usage-limits.md)・[ADR-0016](../adr/0016-mcp-response-slot-lifetime.md) の共有枠と応答完了までの寿命
- [レビュー分析仕様](../specs/review-analysis.md) と [ADR-0011](../adr/0011-client-review-analysis-guide.md) の埋め込みガイドの契約
- [開発手順](../development.md) と [ADR-0001](../adr/0001-runtime-and-deployment.md)・[ADR-0002](../adr/0002-conventions-tooling.md)・[ADR-0003](../adr/0003-go-format-and-lint.md) の実行・検査手順

## 関連 Issue

Closes #77
