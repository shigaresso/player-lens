# MCP のツール定義と実行 handler をサーバー生成から分離した

## 背景・目的

[Issue #50](https://github.com/shigaresso/player-lens/issues/50) に対応し、`newServer` に混在していたツールの公開定義と実行処理を読み分けられる構成にした。

最新の origin/main（60c6e1425eb7e1315d4bfa9add7430b5cf945c52）から `codex/50-mcp-tool-boundaries` を作成した。以前のブランチは維持し、upstream は設定していない。push・PR 作成・Issue のクローズをせず、ローカル確認用の状態にする。この文書は後続の PR 本文にも使用できる。

## 変更内容

- `collection_definition.go` へツール名・説明・入出力 schema・annotations を移し、`collectReviewsDefinition` で生成する形にした。公開内容と生成ごとの可変データの独立性を維持した。
- `collection_handler.go` へ `newCollectReviewsHandler` を分離した。HTTP と SDK のキャンセルを収集へ伝え、終了時に連携 callback を停止して所有 context を解放する処理を維持した。
- `newServer` は Server の生成、対応プロトコルの設定とツール登録を担当する構成にした。HTTP ごとの Server と共有 limiter の寿命、`collect` の実行手順は維持した。
- 分離した定義の独立性、実行 handler のキャンセル伝播と終了時の解放を直接確認するユニットテストを追加した。
- MCP 仕様とユースケースの文書へ責務の境界を記載した。

## 検証

### 抽出前の確認

`docker compose run --rm --no-deps -T verify go test -count=1 ./internal/reviewmcp` と、`go test -tags=integration -race -count=1 ./internal/reviewmcp -run 'Test(MCPProtocol|HTTPRequestCancellation|.*Usage.*|.*Limit.*)'` を同じ verify サービスで実行して成功した。後者は SDK の対応4版・HTTP 切断・共有の開始回数などの関連契約を確認した。全結合テストの実行とは区別する。

### 抽出後の確認

抽出後に同じユニットテストと、`docker compose run --rm --no-deps -T verify go test -tags=integration -race -count=1 ./internal/reviewmcp` が成功した。後者は対象パッケージの通常テストと全結合テストを含む。対応4版の一覧・呼び出し・HTTP 切断、公開側を含む共有予算・同時枠、結果と分析ガイドを確認した。

既存テストと期待結果は変更していない。動作を変えない抽出のため形式上の Red は作らず、抽出と対応文書を 90f1440 にまとめた。直接テストは別の変更単位として追加した。

### 直接テスト

各責務のテスト追加後に、同じ verify サービスで関連テストを順に実行し、抽出済みの実装で Green になることを確認した。プロダクションコードをテストに合わせて変更せず、形式上の Red は作っていない。

- `go test -count=1 ./internal/reviewmcp -run '^TestCollectReviewsDefinition$'`：4状況で、ツール名・説明、annotations、入れ子と必須項目を含む入出力 schema を変更しても別の定義へ伝播しないことを確認した。公開内容そのものは既存 SDK 結合テストで引き続き確認した。
- `go test -race -count=1 ./internal/reviewmcp -run '^TestCollectReviewsHandlerCancellation$'`：2状況で、HTTP／SDK のそれぞれのキャンセルが取得へ届き、公開コード `cancelled` と元の原因が保持されることを確認した。HTTP の context の値と、もう一方の親をキャンセルしないことも確認した。テスト自身の goroutine は失敗時にも取得を解除して完了を待つ。
- `go test -race -count=1 ./internal/reviewmcp -run '^TestCollectReviewsHandlerCleanup$'`：正常・不正入力・取得失敗の3状況で、終了後に両親への連携 callback が残らず、取得へ渡した context が解放されることを確認した。`context.AfterFunc` の登録・停止を観測するテスト用 context を使い、停止しない defer の欠落を検出できるようにした。実際のキャンセル伝播は前述の標準 context と既存 HTTP 切断テストで確認した。

### 全体の検証

固定版 Compose の `bash scripts/check-go.sh format-fix` で新しいテストの整形を揃えた後、`bash scripts/check-app.sh` が成功した。整形・go vet・通常テスト全7パッケージ、integration タグを含む lint（0 issues）、Git 非依存3条件、race 付き通常・結合テスト全7パッケージを確認した。

同じ本番用 image ID sha256:0e7a0efc81862403be9907dab6476ef3a7ed0fa6bf214a1c748d54e14ee59553 で、ローカル／公開設定の HTTP/MCP、Host・Origin 拒否、PORT、公開 URL と利用制限の不正設定の非0終了、SIGTERM/SIGINT の正常終了を確認した。実 Steam・実 LLM へは接続していない。ChatGPT の手動接続と配置は、公開動作を変えないローカルの責務整理のため今回実行していない。

### フックと検証環境

90f1440 の必須 pre-commit（規約ツール全177テスト・静的検査・index の Go 整形）と commit-msg が成功した。追加のコミットも必須フックを通常どおり実行する。検査ツール・Docker・CI・フック設定は、前回検証した 2813e78 と今回の起点の差分がないことを確認した。隔離した実フックの独立検証は、同じ作業環境で前回成功した全8テストの結果を再利用し、今回実行した検査とは区別する。

push 前のため今回のコード CI と、PR 作成 CLI の送信済み head/base を用いた検査は未実施。ローカルのコミット範囲と PR 用本文を検査し、後続の push・PR 作成でコード CI を確認する。coverage の閾値・分岐・mutation（Stryker を含む）は現行リポジトリに未導入であり、既存検査の成功を品質目標の達成とは扱わない。

### AI のセルフレビュー

ツールの説明・annotations・入出力 schema をそのまま移し、`context.WithCancel`・`context.AfterFunc`・defer の順序と `collect` の引数も維持したことを差分で確認した。`NewHandler` の検査・SDK 設定・limiter の生成と共有、`collect` の入力検査・受付・期限・返却には差分がない。人によるレビューは未実施。

抽出前の公開定義・schema・context 連携の本文と移動先が一致することを読み取りスクリプトでも確認した。直接テストは実ソケットを起動せずに実行し、SDK・HTTP の実接続の結合テストを置き換えていない。整形後の追加差分が新しいテストと作業記録だけであることを確認した。

## 参照仕様・ADR

- docs/specs/review-mcp.md「接続と入力」「結果・失敗・資源」「ツール登録と実行の境界」「呼び出し回数と同時数」「デバッグと検証」と ADR-0007「決定」
- docs/specs/mcp-usage-limits.md「目的と適用範囲」「受付と固定窓」と ADR-0008「決定」
- docs/specs/review-analysis.md「提供する契約」「保証する範囲」と ADR-0011「決定」
- docs/use-cases.md「パーツとの境界」
- docs/development.md のフック・アプリケーション検査・Issue／PR 作成手順、ADR-0001・0002・0003

## 関連 Issue

Closes #50
