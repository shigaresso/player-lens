# MCP のツール定義と実行 handler をサーバー生成から分離した

## 背景・目的

[Issue #50](https://github.com/shigaresso/player-lens/issues/50) に対応し、`newServer` に混在していたツールの公開定義と実行処理を読み分けられる構成にした。

最新の origin/main（60c6e1425eb7e1315d4bfa9add7430b5cf945c52）から `codex/50-mcp-tool-boundaries` を作成した。以前のブランチは維持し、upstream は設定していない。push・PR 作成・Issue のクローズをせず、ローカル確認用の状態にする。この文書は後続の PR 本文にも使用できる。

## 変更内容

- `collection_definition.go` へツール名・説明・入出力 schema・annotations を移し、`collectReviewsDefinition` で生成する形にした。公開内容と生成ごとの可変データの独立性を維持した。
- `collection_handler.go` へ `newCollectReviewsHandler` を分離した。HTTP と SDK のキャンセルを収集へ伝え、終了時に連携 callback を停止して所有 context を解放する処理を維持した。
- `newServer` は Server の生成、対応プロトコルの設定とツール登録を担当する構成にした。HTTP ごとの Server と共有 limiter の寿命、`collect` の実行手順は維持した。
- MCP 仕様とユースケースの文書へ責務の境界を記載した。

## 検証

### 抽出前の確認

`docker compose run --rm --no-deps -T verify go test -count=1 ./internal/reviewmcp` と、`go test -tags=integration -race -count=1 ./internal/reviewmcp -run 'Test(MCPProtocol|HTTPRequestCancellation|.*Usage.*|.*Limit.*)'` を同じ verify サービスで実行して成功した。後者は SDK の対応4版・HTTP 切断・共有の開始回数などの関連契約を確認した。全結合テストの実行とは区別する。

### 抽出後の確認

抽出後に同じユニットテストと、`docker compose run --rm --no-deps -T verify go test -tags=integration -race -count=1 ./internal/reviewmcp` が成功した。後者は対象パッケージの通常テストと全結合テストを含む。対応4版の一覧・呼び出し・HTTP 切断、公開側を含む共有予算・同時枠、結果と分析ガイドを確認した。

既存テストと期待結果は変更していない。動作を変えない抽出のため形式上の Red は作らず、抽出と対応文書を1つのコミットにする。直接テストは別の変更単位として追加する。

### AI のセルフレビュー

ツールの説明・annotations・入出力 schema をそのまま移し、`context.WithCancel`・`context.AfterFunc`・defer の順序と `collect` の引数も維持したことを差分で確認した。`NewHandler` の検査・SDK 設定・limiter の生成と共有、`collect` の入力検査・受付・期限・返却には差分がない。人によるレビューは未実施。

## 参照仕様・ADR

- docs/specs/review-mcp.md「接続と入力」「結果・失敗・資源」「ツール登録と実行の境界」「呼び出し回数と同時数」「デバッグと検証」と ADR-0007「決定」
- docs/specs/mcp-usage-limits.md「目的と適用範囲」「受付と固定窓」と ADR-0008「決定」
- docs/specs/review-analysis.md「提供する契約」「保証する範囲」と ADR-0011「決定」
- docs/use-cases.md「パーツとの境界」
- docs/development.md のフック・アプリケーション検査・Issue／PR 作成手順、ADR-0001・0002・0003

## 関連 Issue

Closes #50
