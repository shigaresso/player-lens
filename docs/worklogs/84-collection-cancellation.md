# 失敗結果の生成を共通化し MCP の開始前キャンセルを早期終了した

## 背景・目的

[Issue #84](https://github.com/shigaresso/player-lens/issues/84) に対応した。開始前にキャンセル済みでも下位の収集処理へ進んで失敗結果を生成する構成から、MCP 用 UseCase の先頭で失敗結果を応答処理へ渡し、早期 return する構成にした。

PR #83 の統合を確認し、最新 origin/main の f1444e6ec89b0e5ec983a2e92201e7a76932e8c0 から codex/84-collection-cancellation を作成した。既存のブランチと作業を保持し、upstream は設定していない。push・PR 作成・Issue のクローズは行っていない。

## 変更内容

- steam/domain.NewFailedCollection は、渡された Collection の App ID・条件・取得済みレビュー・成功ページ数を保持し、終了理由を StopFailed としたコピーを生成する。レビュー配列を複製し、nil／空配列も維持する。値の検査・既定値の適用・context・原因の分類は行わない。
- CollectionState.Fail とカーソル循環の失敗確定を共通生成へ接続した。部分結果を空の失敗結果で置き換えず、終了済み状態の不変性と終了理由の優先順を維持した。
- MCP 用 CollectReviews は開始前キャンセル／期限超過を入口で確認し、共通生成と respond で終了する。下位の収集・期限の生成・利用枠の確保へ進まない。受付拒否の失敗結果にも共通生成を使い、従来の nil 配列を維持した。開始前キャンセルは従来どおり非 nil の空配列を維持した。
- 元のキャンセル／期限超過の原因を errors.Is で識別できるよう保持し、診断の文脈を MCP 用 UseCase の開始前終了へ変更した。公開コードや説明文は変更していない。
- 利用枠の確保・defer による返却、8秒の期限、同期的な応答変換は application に維持した。Controller へ管理を移していない。収集失敗は Result.Err、応答処理の失敗は戻り error という契約も維持した。

## 検証

### Red・Green とコミット

| 対象 | 開発と確認 | コミット |
| --- | --- | --- |
| 共通の失敗結果生成 | TestNewFailedCollection だけを追加し、未実装 API のコンパイル失敗を Red として確認した。生成処理を実装し、同じテストの Green を確認した | 153f157 |
| 開始前終了の回帰契約 | 既存のテストをキャンセル／期限超過のテーブルへ拡充した。入力条件・元の原因・空配列・応答一度・非取得・時計の非参照・枠の非消費を確認した。実装変更前に Green だったため、テストのみで確定した | 34fbcb9 |
| 早期 return と共通生成への接続 | 回帰テストの期待値を変更せず、関連5パッケージの通常テストが成功した。空行規則の指摘を修正した。開始前に下位の収集を呼ばない構造はコードレビューでも確認した | 8428238 |

主なコマンドは Docker Compose verify の `go test -count=1 ./internal/steam/domain -run '^TestNewFailedCollection$'`、`go test -count=1 ./internal/reviewmcp/application -run '^TestCollectReviewsCancelledBeforeStart$'`、`go test -count=1 ./internal/steam/domain ./internal/steam/application ./internal/reviewmcp/application ./internal/reviewmcp/adapter/handler/...`。構造変更のためだけの Red は作っていない。

### 全体検証

- `bash scripts/check-app.sh` が成功した。gofmt・vet・全通常テスト、lint 0 issues、Git 非依存3条件、race 付き通常／結合テストを確認した。結合テストだけを実行したとは扱わない。
- 同じ本番 image ID `sha256:9164b9e130a07768646a0161754b84b4723aa24fd3261dc93beb5ed1d4d5e98f` で、HTTP/MCP、ローカル／公開 Host・Origin、PORT、不正設定、SIGTERM/SIGINT を確認した。実 Steam への取得は行っていない。
- 既存154個の Test 関数を保持し、共通生成の契約テスト1個を追加して155個となった。空／部分失敗・循環・終了後不変、変換中の枠保持、拒否、変換失敗、panic、キャンセルと短い期限を確認する既存テストを維持した。
- 各コミットの必須フック（規約用 Python 177テスト、静的検査、整形検査、コミットメッセージ検査）が成功した。最終のコミット範囲・差分・文書リンク・Issue/PR 用本文の検査結果は Issue に記録した。

### AI のセルフレビューと制約

開始前終了の早期 return、同時枠の確保と返却の対応、同期的な応答変換の範囲、部分結果と元の原因の保持を確認した。新しい通信・時計・goroutine・再試行は追加していない。失敗結果の独立性のため、CollectionState の失敗確定時にもレビュー配列のコピーが発生する。既存の Result によるスナップショット生成の契約も維持した。人によるレビューは未実施。

GitHub 認証と Docker は権限付きで実行し、フック・対象・閾値を省略／緩和していない。push 前のため pre-push・送信済み head/base による PR 作成検査・コード CI は未実施。Render 配置・実外部サービス接続、coverage 閾値・分岐・mutation（Stryker を含む）は未導入／未実施であり、全検証成功を100%の保証として扱わない。

## 参照仕様・ADR

- [データ・調査仕様](../specs/data-and-research.md) のジョブ・上限・部分失敗
- [収集仕様](../specs/collect-reviews.md) の収集状態・失敗結果の共通生成・期限と失敗、[ADR-0005](../adr/0005-bounded-review-collection.md)・[ADR-0014](../adr/0014-steam-collection-state.md)・[ADR-0015](../adr/0015-collection-options-boundary.md)
- [MCP 利用制限](../specs/mcp-usage-limits.md) の受付と固定窓・拒否応答、[ADR-0008](../adr/0008-mcp-usage-limits.md)
- [MCP 仕様](../specs/review-mcp.md) の結果・失敗・資源、[ADR-0007](../adr/0007-review-mcp.md)・[ADR-0013](../adr/0013-domain-folder-layers.md)・[ADR-0016](../adr/0016-mcp-response-slot-lifetime.md)
- [操作一覧](../use-cases.md)、[開発手順](../development.md) の Docker・テスト・必須フック

## 関連 Issue

Closes #84
