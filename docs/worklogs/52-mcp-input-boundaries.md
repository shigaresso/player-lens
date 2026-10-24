# MCP 入力の JSON 読み取りと収集条件の構築を分離した

## 背景・目的

[Issue #52](https://github.com/shigaresso/player-lens/issues/52) に対応し、`parseInput` に同居していた入力形式の検査と収集条件の構築を分離した。

最新の origin/main（0d19c42527b8fc76dbca1cb86a805df6dea79e79）から `codex/52-mcp-input-boundaries` を作成した。以前のブランチは維持し、upstream は設定していない。push・PR 作成・Issue のクローズは行わず、ローカル確認用の状態にする。この文書は後続の PR 本文にも使用できる。

## 変更内容

- `input_decode.go` の `decodeToolInput` が JSON の形式を検査して `toolInput` を返し、読み取り段階で既定値や MCP の上限を適用しない構成にした。
- `input_options.go` の `normalizeToolInput` が MCP の既定値・上限を適用し、評価種別の変換と共通正規化により収集条件を作る構成にした。
- `parseInput` は両処理を順に呼び、既存の正常結果と失敗時のゼロ値・`ErrInvalidArgument` を保持するエラーを返す境界にした。
- MCP 仕様とユースケースの文書へ責務の境界を記載した。

## 検証

### 抽出前の確認

`docker compose run --rm --no-deps -T verify go test -count=1 ./internal/reviewmcp` が成功した。既存の入力・評価種別・結果・利用制限を対象とし、抽出前の正常結果と失敗時の振る舞いを確認した。

### 抽出後の確認

抽出後にも同じ MCP ユニットテストが成功した。既存テストと期待結果は変更していない。動作を変えない抽出のため形式上の Red は作らず、責務分離と対応文書を1つのコミットにまとめる。直接回帰テストは別の変更単位で追加する。

### AI のセルフレビュー

JSON の検査順序・toolInput の型とタグ・正確な項目名と null の検査・末尾の検査を維持し、後半の App ID・MCP 上限・既定値・評価種別・共通正規化も同じ順序で移したことを確認した。`parseInput` が両段階の失敗を既存と同じゼロ値・文言・原因へ変換することを確認した。`collect` の入力検査と利用枠の消費の順序、schema の内容、収集・CLI・依存モジュールには変更がない。人によるレビューは未実施。

## 参照仕様・ADR

- docs/specs/review-mcp.md「接続と入力」「入力変換の境界」「結果・失敗・資源」「デバッグと検証」と ADR-0007「決定」
- docs/specs/collect-reviews.md「入力と正規化」、docs/specs/data-and-research.md の導入済み収集操作と ADR-0012「決定と境界」
- docs/specs/mcp-usage-limits.md「受付と固定窓」と ADR-0008「決定」
- docs/use-cases.md「パーツとの境界」
- docs/development.md のフック・アプリケーション検査・Issue／PR 作成手順と ADR-0001〜0003

## 関連 Issue

Closes #52
