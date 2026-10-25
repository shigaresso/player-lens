# レビュー収集の状態と更新ルールを domain に分離した

## 背景・目的

[Issue #58](https://github.com/shigaresso/player-lens/issues/58) に対応した。CollectReviews が取得・期限・キャンセルと結果・重複・カーソル・終了条件を同じ関数で管理していたため、状態と更新ルールを steam/domain にまとめ、application を取得の手順に整理した。

最新 origin/main（a9ec35e963d2a6b8b771ae77e7bf0f25bc4b6e93、PR #57 統合済み）から `codex/58-steam-collection-state` を作成した。既存ブランチは維持し、upstream は設定していない。push・PR 作成・Issue のクローズをせず、ローカルで確認できる状態にする。

## 変更内容

- CollectionState が Collection・既読 ID・既読カーソル・次のカーソルを非公開で所有する構成にした。NewCollectionState が App ID と条件を検査・正規化する。
- NextPage が残件数に応じた条件と続行可否、AddPage が成功数・重複排除・終了判断、Fail が失敗確定を担当する。空ページ、件数上限、循環、ページ上限の順序と、循環したページも部分結果に含める契約を維持した。
- 終了後の状態を変更せず、Result はレビュー配列を含めてコピーする。取得依存・context・時計・再試行を domain に持たせていない。
- CollectReviews は取得依存を確認し、状態の生成、2分の期限、ページ取得と反映、失敗の文脈・部分結果の返却を担当する。App ID と条件の検査を生成側に集約した。App ID が0のエラーには生成の文脈が加わるが、ゼロ値と ErrInvalidArgument による識別は維持した。
- ErrCursorCycle を domain へ移し、application と MCP のテストの参照を更新した。互換エイリアスやラッパーは残していない。
- 収集仕様・ユースケース・文書一覧と関連 ADR を更新し、ADR-0014 に状態の契約・コピーの費用・application との境界を記録した。

## 検証

### Red・Green とコミット

変更前に `docker compose run --rm --no-deps -T verify go test -count=1 ./internal/steam/...` で3パッケージの既存テストが成功した。以下の契約を順に追加して、それぞれの理由で Red を確認してから実装した。同じ契約の状況はテーブル駆動でまとめた。

| 対象テスト | 契約 | 確認した Red |
| --- | --- | --- |
| TestNewCollectionState | 既定値・指定条件・残件数、App ID と条件の拒否 | NewCollectionState 未定義 |
| TestCollectionStatePages | 成功数・短いページ・重複・最初の値と順序・正常終了 | AddPage 未定義 |
| TestCollectionStateCursor | 循環の原因識別と終了優先順位 | domain.ErrCursorCycle 未定義 |
| TestCollectionStateFail | 先頭・途中失敗の結果と条件の保持 | Fail 未定義 |
| TestCollectionStateStopped | 各終了理由での結果の不変性 | 終了後の追加でページ数・レビューが変化した |
| TestCollectionStateResultOwnership | 進行中・成功・失敗の戻り値の所有 | 返したレビュー配列の変更が内部状態へ漏れた |

生成・ページ・循環・Fail の Red は `go test -count=1 ./internal/steam/domain -run '^対象テスト名$'` を上記 verify サービスで実行した。終了後・所有権も同じ方法で検出した。実装後に domain 全体を実行して Green を確認し、最終的に6関数・29状況で状態の契約を確認した。

エラー移動に必要な既存の参照を更新した後、`go test -count=1 ./internal/steam/... ./internal/reviewmcp/adapter` が4パッケージで成功した。途中で重複した import は修正し、期待値は変更していない。状態と直接の契約・エラー移動・仕様を dc38a36 にまとめた。

その後、既存テストを変更せず CollectReviews を状態に従う手順へ整理した。`go test -count=1 ./internal/steam/application ./internal/reviewmcp/application` が成功し、1902bc4 に分けた。ここは動作を維持する Refactor とし、形式上の Red を作っていない。両コミットの必須 pre-commit（規約ツール177テスト・静的検査・index の整形）と commit-msg が成功した。

### 全体の検証

`bash scripts/check-app.sh` が終了コード0で成功した。整形・vet・通常テスト、integration タグを含む lint（0 issues）、Git 非依存の3条件、race 付き通常／結合テストを確認した。通常は全15パッケージを収集して14で実行し、cmd/collect-reviews の root は integration タグのコマンド実行テストで検証した。race 付き通常／結合は全15パッケージで成功した。

本番用 image ID `sha256:0d854fe7d83e97496643f24426a953cd202386e5802df3bc3d8d36f154b8d154` を一度ビルドし、同じイメージでローカル／公開設定の HTTP/MCP、Host・Origin の拒否、PORT、非 root・読み取り専用・ソース共有なしの起動、不正設定の非0終了、SIGTERM/SIGINT の正常終了を確認した。

### 実行環境の中断と復旧

途中で利用上限により権限レビューを実行できず、終了後のテストは未実行のまま中断した。ユーザーの再開指示後に同じ権限付き実行を再開した。Docker が停止していたため起動し、次に公開 registry の認証ヘルパーが応答せずビルドが停止した。

この作業で停止中の Docker クライアントを終了し、一時設定 `/private/tmp/player-lens-58-docker-config`（空の auths と既存 CLI plugin への参照）で公開イメージへ匿名接続した。既存の Docker 設定と認証情報は変更していない。以後の Docker とコミットフックは `DOCKER_CONFIG=/private/tmp/player-lens-58-docker-config DOCKER_HOST=unix:///Users/soregashi/.docker/run/docker.sock` を付けて同じ固定版のビルド・検査を実行した。検査やフックは省略していない。

### AI のセルフレビュー

既存124テスト関数内の文字列・数値リテラルが origin/main とすべて一致することを読み取りスクリプトで確認した。これは既存の期待値を変えていないことの補助確認であり、実際の契約は実行したテストで確認した。

終了優先順位・成功ページ数・重複と最初の値・残件数・循環ページの部分結果、入力拒否とエラー原因を読み、抽出前と同じであることを確認した。application は終わった状態を先に確認してから context を検査し、最終ページで終了条件を満たした場合の優先順位を維持した。

domain は I/O と context を持たず、application の期限・ページ間キャンセル・原因のラップ・defer による解放は維持した。Result のコピーは返却時に一度だけ取得し、ページごとの全レビューコピーは追加していない。既存の CLI・MCP の JSON・上限・ガイド・保存処理・HTTP adapter は変更していない。人によるレビューは未実施。

### 未実施と保証範囲

push 前のためコード CI と、PR 作成 CLI の送信済み head/base を使う検査は未実施で、CI の完了待機もしない。実 Steam・LLM・ChatGPT 接続・配置も今回実施していない。coverage 閾値・分岐・mutation（Stryker を含む）は未導入であり、今回の検査成功をそれらの品質目標達成とは扱わない。

## 参照仕様・ADR

- docs/specs/collect-reviews.md「入力と正規化」「ページの走査と結果」「収集状態の境界」「期限と失敗」「検証と未導入項目」、data-and-research.md のレビュー収集・部分失敗、ADR-0005・0012・0013・0014。
- docs/specs/review-cli.md と review-mcp.md の既存の結果・失敗と資源の契約、ADR-0006・0007。CLI・MCP の入口と保存先の振る舞いを結合で確認した。
- docs/use-cases.md のパーツとの境界、docs/development.md の検証・フック・本文検査、ADR-0001〜0003 の既存の実行・検証環境。

## 関連 Issue

Closes #58
