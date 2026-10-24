# MCP のローカル HTTP 保護を mcpaccess へ分離した

## 背景・目的

[Issue #48](https://github.com/shigaresso/player-lens/issues/48) に対応し、MCP handler の組み立てと HTTP の受付判断を分離した。

最新の origin/main（d33286474e44ed82755b05eefbe2d2f92310820e）から codex/48-local-mcp-protection を作成した。前回の作業ブランチは維持し、今回の upstream は設定していない。push・PR 作成・Issue のクローズは行わず、ローカル確認用の状態にする。この文書は後続の PR 本文にも使用できる。

## 変更内容

- reviewmcp.NewHandler に直接置かれていたローカル用の Host・cross-origin 保護と localHost を、既存の mcpaccess パッケージへ移した。
- ProtectLocal の入力は有効な非 nil の HTTP handler とし、Host、cross-origin、next の順に処理する契約を明記した。既存の判定と Host 拒否時の403・説明文を維持した。
- NewHandler は公開設定に応じて ProtectLocal と Access.Protect を選ぶ構成にした。SDK の設定、共有 limiter の所有、公開側の authority 正規化、CLI と収集操作は変更していない。
- ProtectLocal の公開契約を、後続 handler の実行回数・応答と拒否の優先順で直接確認するユニットテストを追加した。NewHandler 経由の既存テストも維持した。
- 公開 MCP 仕様・MCP 仕様・ADR-0009・ユースケースの責務の説明を更新した。

## 検証

### 抽出前後の確認

抽出前と抽出後に `docker compose run --rm --no-deps -T verify go test -count=1 ./internal/mcpaccess ./internal/reviewmcp` が成功した。既存の期待結果は変更せず、NewHandler 経由の Host・Origin・本文上限と、公開側の保護を確認した。動作を維持する Refactor のため形式上の Red は作っていない。

最初のコミット 8884e09 で抽出と対応する仕様をまとめた。ProtectLocal の直接テストは、プロダクションコードを変更せず、抽出とは別に確定する単位とした。

### ProtectLocal の直接テスト

外部テストパッケージから、MCP SDK・収集依存・実ソケットを使わずに ProtectLocal を呼ぶ TestProtectLocal を追加した。17状況の名前付きテーブルで、localhost・IPv4/IPv6 loopback・一致する Origin の許可、空/外部 Host と転送ヘッダーの迂回拒否、異なる Origin と cross-site の拒否を確認した。許可時の next の1回実行とヘッダー・ステータス・本文の維持、拒否時の非実行、Host 拒否の説明文と Origin より先に返す順序も確認した。

`docker compose run --rm --no-deps -T verify go test -count=1 ./internal/mcpaccess -run '^TestProtectLocal$'` が、抽出済みの実装で Green だった。テスト追加に合わせたプロダクションコードの変更や、形式上の Red は作っていない。

### 全体の検証

固定版 Compose の `bash scripts/check-go.sh format-fix` で新しいテストの整形を揃えた後、`bash scripts/check-app.sh` が成功した。整形・go vet・通常テスト全7パッケージ、integration タグを含む lint（0 issues）、Git 非依存3条件、race 付き通常・結合テスト全7パッケージを確認した。

同じ本番用 image ID sha256:c3121e679b3e47a9a9fd6e200fad654b19607d0481a77173d6953713d3355bbc で、ローカル／公開設定の HTTP/MCP、Host・Origin の拒否、PORT、公開 URL と利用制限の不正設定の非0終了、SIGTERM/SIGINT の正常終了を確認した。実 Steam・実 LLM へは接続していない。配置用の公開は行っていない。

### フックと検証環境

8884e09 の必須 pre-commit（規約ツール全177テスト・静的検査・index の Go 整形）と commit-msg が成功した。直接テストのコミットでも通常どおり実行する。検査ツール・Docker・CI・フック設定は前回検証した状態と同じであることを、2813e78 と今回の起点の差分で確認した。隔離した実フックの独立検証は、同じ作業環境で前回成功した全8テストの結果を再利用する。今回再実行した検査とは区別する。

push 前のため今回のコード CI と、PR 作成 CLI の送信済み head/base を用いた検査は未実施。coverage・分岐・mutation（Stryker を含む）は未導入であり、既存検査の成功を品質目標の達成とは扱わない。

### AI のセルフレビュー

ローカルの Host 判定を同じ処理のまま移し、Host 拒否の文言と cross-origin より先に検査する順序を維持したことを確認した。公開用の実装、SDK 設定と共有 limiter の生成場所に差分がないことも確認した。人によるレビューは未実施。

直接テストが MCP SDK・Steam の依存を必要とせず、許可時の後続の応答維持と拒否時の非実行を観測することを確認した。Host と Origin の両方が不許可の入力では Host の説明文を確認するため、順序の変更も検出できる。整形後の追加差分がテストと対応する文書だけであることも確認した。

## 参照仕様・ADR

- docs/specs/mcp-public-access.md「HTTP の受付」「保護の組み立ての境界」「資源・共有範囲」「検証」と ADR-0009「決定」
- docs/specs/review-mcp.md「接続と入力」「結果・失敗・資源」「呼び出し回数と同時数」と ADR-0007「決定」
- docs/specs/mcp-usage-limits.md「目的と適用範囲」「受付と固定窓」と ADR-0008「決定」
- docs/use-cases.md「パーツとの境界」
- docs/development.md「各フックの責務」「Go・Compose・本番用イメージの検査」「Issue・PR の作成手順」、ADR-0001・0002・0003

## 関連 Issue

Closes #48
