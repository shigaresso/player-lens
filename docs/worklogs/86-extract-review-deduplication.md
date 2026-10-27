# ページ反映の手順からレビューの重複排除を分離した

## 背景・目的

[Issue #86](https://github.com/shigaresso/player-lens/issues/86) に対応した。CollectionState.AddPage の手順から既読 ID の確認・登録とレビュー配列への追加を分離し、ページ反映と終了判断の流れを読み取りやすくした。

PR #85 の統合を確認し、最新 origin/main の 2b8e8cfa62d015614106c053d769075153347e59 から codex/86-extract-review-deduplication を作成した。既存のブランチとコミットは保持した。upstream は設定せず、push・PR 作成・Issue のクローズは行っていない。

## 変更内容

- 同じ CollectionState の非公開メソッド addUniqueReviews へ、既存の重複排除ループを移した。既読 ID の確認・登録と配列への追加を一緒に管理した。
- AddPage は既存と同じ位置で一度だけ呼び出す。空ページ・件数・循環・ページ数の終了判定、成功ページ数とカーソルの更新、終了済み状態の扱いを維持した。
- 最初の本文・全項目・取得順、ID 0 の重複排除、重複だけのページの継続と部分結果を維持した。
- 公開 API・新規パッケージ・interface・依存・I/O を増やしていない。カーソルの型・サイズ制限、Steam 応答のサイズ制限、ログ・観測情報、UseCase・SDK は変更していない。
- 収集仕様・ADR-0014・操作一覧へ、手順と具体的な処理の分離を記録した。

## 検証

### Refactor の前後

動作を変更しない整理のため、形式のための Red や private メソッドをなぞるテストは追加していない。Docker Compose verify で `go test -count=1 ./internal/steam/domain ./internal/steam/application` を実行し、変更前・変更後の両方で Green を確認した。

既存のテスト本体・入力・期待値は変更・追加・削除していない。TestCollectionStatePages、TestCollectionStateCursor、TestCollectionStateStopped、TestCollectionStateResultOwnership と共通収集のテストで、最初の値・順序・ID 0・重複だけのページ・終了優先順・循環・部分結果・返却したコピーの独立性を確認した。

本番コードの差分を照合し、既存ループの移動と、元の位置への呼び出しの追加以外に変更がないことを確認した。実装と対応仕様を d70f4d8 でコミットした。

### 全体検証

- `bash scripts/check-app.sh` が成功した。gofmt・vet・全通常テスト、lint 0 issues、Git 非依存3条件、race 付き通常／結合テストを実行した。結合テストだけの成功とは扱わない。
- 同じ本番 image ID `sha256:5c52b6988bcb36ed3c366ad89a320b829e914d85daa14a074e5f73da0d4e6633` で HTTP/MCP、ローカル／公開 Host・Origin、PORT、不正設定、SIGTERM/SIGINT を確認した。
- 実装コミットで必須フックの規約用 Python 177テスト・静的検査・整形検査・メッセージ検査が成功した。文書コミットのフックと最終の range・差分・文書リンク・Issue/PR 用本文の結果は Issue に記録した。

### AI のセルフレビューと未実施項目

終了済み状態を先に拒否し、空ページの終了後・件数上限の判定前に重複排除を呼ぶ位置を維持した。分離先は既読 ID と配列の両方を更新し、状態を呼び出し側へ露出させていない。終了条件の優先順を手順内で読めることも確認した。人によるレビューは未実施。

開始時は Docker 停止のため検証を実行できず、Docker Desktop を起動してから変更前の確認を行った。GitHub 認証・Docker・必須フックは権限付きで実行した。対象や設定を省略・緩和していない。

push 前のため pre-push・送信済み head/base の PR 作成検査・コード CI は未実施。実 Render 配置・実 Steam／LLM 接続、coverage 閾値・分岐・mutation（Stryker を含む）は未導入／未実施であり、検査成功を100%の保証として扱わない。

## 参照仕様・ADR

- [データ・調査仕様](../specs/data-and-research.md) のレビュー・上限・部分失敗
- [収集仕様](../specs/collect-reviews.md) のページの走査と結果・収集状態の境界、[ADR-0005](../adr/0005-bounded-review-collection.md) の上限・部分結果
- [ADR-0013](../adr/0013-domain-folder-layers.md) の domain 内の責務と依存、[ADR-0014](../adr/0014-steam-collection-state.md) の状態・重複・終了と取得手順の境界
- [操作一覧](../use-cases.md)、[開発手順](../development.md) の Docker・通常／結合テスト・必須フック

## 関連 Issue

Closes #86
