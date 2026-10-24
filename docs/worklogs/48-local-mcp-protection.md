# MCP のローカル HTTP 保護を mcpaccess へ分離した

## 背景・目的

[Issue #48](https://github.com/shigaresso/player-lens/issues/48) に対応し、MCP handler の組み立てと HTTP の受付判断を分離した。

最新の origin/main（d33286474e44ed82755b05eefbe2d2f92310820e）から codex/48-local-mcp-protection を作成した。前回の作業ブランチは維持し、今回の upstream は設定していない。push・PR 作成・Issue のクローズは行わず、ローカル確認用の状態にする。この文書は後続の PR 本文にも使用できる。

## 変更内容

- reviewmcp.NewHandler に直接置かれていたローカル用の Host・cross-origin 保護と localHost を、既存の mcpaccess パッケージへ移した。
- ProtectLocal の入力は有効な非 nil の HTTP handler とし、Host、cross-origin、next の順に処理する契約を明記した。既存の判定と Host 拒否時の403・説明文を維持した。
- NewHandler は公開設定に応じて ProtectLocal と Access.Protect を選ぶ構成にした。SDK の設定、共有 limiter の所有、公開側の authority 正規化、CLI と収集操作は変更していない。
- 公開 MCP 仕様・MCP 仕様・ADR-0009・ユースケースの責務の説明を更新した。

## 検証

### 抽出前後の確認

抽出前と抽出後に `docker compose run --rm --no-deps -T verify go test -count=1 ./internal/mcpaccess ./internal/reviewmcp` が成功した。既存の期待結果は変更せず、NewHandler 経由の Host・Origin・本文上限と、公開側の保護を確認した。動作を維持する Refactor のため形式上の Red は作っていない。

最初のコミットで抽出と対応する仕様をまとめ、後続の別コミットで ProtectLocal の公開契約を直接確認するユニットテストを追加する。全体のアプリケーション検査は直接テストを追加した後に行い、実行した結果を追記する。

### フックと検証環境

必須の pre-commit と commit-msg は各コミットで通常どおり実行する。検査ツール・Docker・CI・フック設定は前回検証した状態と同じであることを、2813e78 と今回の起点の差分で確認した。隔離した実フックの独立検証は、同じ作業環境で前回成功した全8テストの結果を再利用する。今回再実行した検査とは区別する。

push 前のため今回のコード CI と、PR 作成 CLI の送信済み head/base を用いた検査は未実施。coverage・分岐・mutation（Stryker を含む）は未導入であり、既存検査の成功を品質目標の達成とは扱わない。

### AI のセルフレビュー

ローカルの Host 判定を同じ処理のまま移し、Host 拒否の文言と cross-origin より先に検査する順序を維持したことを確認した。公開用の実装、SDK 設定と共有 limiter の生成場所に差分がないことも確認した。人によるレビューは未実施。

## 参照仕様・ADR

- docs/specs/mcp-public-access.md「HTTP の受付」「保護の組み立ての境界」「資源・共有範囲」「検証」と ADR-0009「決定」
- docs/specs/review-mcp.md「接続と入力」「結果・失敗・資源」「呼び出し回数と同時数」と ADR-0007「決定」
- docs/specs/mcp-usage-limits.md「目的と適用範囲」「受付と固定窓」と ADR-0008「決定」
- docs/use-cases.md「パーツとの境界」
- docs/development.md「各フックの責務」「Go・Compose・本番用イメージの検査」「Issue・PR の作成手順」、ADR-0001・0002・0003

## 関連 Issue

Closes #48
