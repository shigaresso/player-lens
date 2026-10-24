# MCP 利用制限の生成と検証の責務を集約した

## 背景・目的

[Issue #54](https://github.com/shigaresso/player-lens/issues/54) に対応し、環境変数の読み取りと NewHandler で重複していた上限の検証を、設定の生成時へ集約した。

最新の origin/main（fe2e7f1f70292be61d69f218776891f4114a329a）から `codex/54-mcp-limits-construction` を作成した。前回のブランチは維持し、upstream は未設定。push・PR 作成・Issue のクローズをせず、ローカル確認用の状態にする。この文書は後続の PR 本文にも使用できる。

## 変更内容

- 未検証の `LimitsConfig` と生成済みの `Limits` を区別し、`NewLimits` が正数検査とコピーの保持を担当する構成にした。生成済みのフィールドは非公開とした。
- `Limits.Config()` は設定のコピーを返し、`DefaultLimits` の有効な固定値を維持した。生成側の API は I/O や利用枠の操作を含めていない。
- `readMCPLimits` は従来どおり環境変数の構文・31 bit 範囲を解析して生成 API へ渡す構成にした。旧 Validate は削除し、`NewHandler` は未初期化のゼロ値だけを拒否する形にした。
- 呼び出し元とテストを新しい生成 API に移行した。受付・窓・回数・返却・共有状態の所有と寿命は維持した。
- 利用制限仕様・MCP 仕様・ADR-0008・ユースケースを更新した。

## 検証

### 抽出前の確認

`docker compose run --rm --no-deps -T verify go test -count=1 ./cmd/server ./internal/reviewmcp` が成功し、変更前の設定と利用制限のユニットテストを確認した。

### 生成契約の Red・Green

外部テストパッケージへ TestNewLimits の9状況を先に追加し、同じ verify サービスの `go test -count=1 ./internal/reviewmcp -run '^TestNewLimits$'` が LimitsConfig 未実装によるコンパイル失敗になる Red を確認した。正数・独立した日次・分次、Go の正数と環境変数の範囲の区別、各上限の0・負数と失敗時のゼロ値・原因を対象にした。

生成 API と既存参照の移行後、`go test -count=1 ./cmd/server ./internal/reviewmcp` が成功して Green を確認した。旧 Validate の正数・不正値の状況は生成 API のテストへ移し、既定値と NewHandler のゼロ値の拒否も維持した。環境変数と利用制限の既存テストは生成方法と値の参照を機械的に移行し、入力・上限・受付結果の期待を維持した。

`bash scripts/check-go.sh format-fix` で固定版 Compose による整形を行い、同じ verify の `go test -tags=integration -race -count=1 ./cmd/server ./internal/reviewmcp` が成功した。通常テストに加え、SDK・公開設定・共有予算・同時枠・HTTP 切断・待ち受け前の失敗などの既存結合テストを確認した。

### AI のセルフレビュー

NewLimits が旧 Validate と同じ順序・正数条件・エラー原因を保持し、環境変数の解析と起動順序を変更していないことを確認した。NewHandler の残る拒否は外部で作成できる未初期化値に対するものであり、生成済みの各上限の再検査ではない。DefaultLimits は固定の有効値を直接返す。

limiter の本文は設定値の参照先以外で変更がないことを読み取りスクリプトでも確認した。nil の取得依存、収集・JSON 読み取り・CLI・依存モジュールに変更がないことも確認した。人によるレビューは未実施。

## 参照仕様・ADR

- docs/specs/mcp-usage-limits.md「目的と適用範囲」「設定」「設定値の生成と検証」「受付と固定窓」「検証と保証の範囲」、ADR-0008「決定」
- docs/specs/review-mcp.md「呼び出し回数と同時数」と ADR-0007
- docs/specs/public-demo.md、docs/specs/data-and-research.md の実行制限と導入済み収集の範囲
- docs/use-cases.md「パーツとの境界」、docs/development.md、ADR-0001〜0003 の既存の検証環境と規約

## 関連 Issue

Closes #54
