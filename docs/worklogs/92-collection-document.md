# 収集結果の JSON 出力データ構築とエンコードを分離した

## 背景・目的

[Issue #92](https://github.com/shigaresso/player-lens/issues/92) に対応した。MCP と JSON Saver が共通で使う MarshalCollection から出力データの構築を分離し、入力と出力項目の対応を名前で読める構成にした。

PR #91 統合済みの最新 origin/main（3f851b16386750c39dd6f90cafa17729627ed0b1）から codex/92-collection-document を作成した。既存のブランチとコミットは保持した。upstream は設定せず、push・PR 作成・Issue のクローズは行っていない。

## 変更内容

- 同じ jsonoutput/adapter の collection_document.go に非公開の出力型と newCollectionDocument をまとめた。評価種別・レビュー ID の文字列化、空レビューの配列化と出力データの構築を担当する。
- レビューと取得条件の初期化に項目名を明示し、宣言順と照合せずに対応を読めるようにした。1項目ごとの helper や公開 DTO は追加していない。
- MarshalCollection は構築と JSON 化を順に呼ぶ。schema_version=1、すべての項目・値・順序、入力の非変更、未知の評価種別の原因、公開入口の nil と既存のエラー文脈を維持した。
- MCP と Saver の呼び出し元、writer・context・利用枠の所有を変更していない。正規化・通信・保存・再試行・新しいレイヤーや依存は追加していない。
- 対応仕様・ADR-0006・操作一覧に責務を反映した。

## 検証

### 回帰テストの Green と Refactor

既存の出力テストでは1件のレビューだけを使っていたため、同じテーブルに取得順を確認できる2件の異なるレビューと、非 nil の空配列を用意した。最大 ID・0、異なる本文・言語・推薦・作成／更新時刻の対応を確認する。既存の本文と条件、部分失敗、nil の空配列と入力の非変更の期待は維持した。

本番コードを変更する前に、Docker Compose verify で次のコマンドを実行し、3パッケージの Green を確認した。既存の振る舞いを確認する回帰テストなので Red を作らず、テストだけを e4d1a1e へコミットした。

```sh
go test -count=1 ./internal/jsonoutput/adapter \
  ./internal/reviewmcp/adapter/handler ./internal/reviewmcp/adapter/handler/protocol
```

その後、テストを変えずに本番コードを分離し、同じコマンドが Green を維持した。既存の TestMarshalCollection、ID の数値境界、評価種別と未知値の拒否、Saver の書き込み・失敗と MCP の公開結果・サイズ・原因保持を確認した。分離先をなぞるだけのテストや、テスト用の公開 API は追加していない。

比較スクリプトで、出力型のフィールド順・型・JSON タグ、JSON 化とそのエラー処理の本体が origin/main と同じことも確認した。項目名を付けた対応は差分と公開入口のテストで確認した。

実装と対応仕様を 80f486b にコミットした。回帰テストと実装の両コミットで、必須 pre-commit の Python 177テスト・静的検査・index 整形検査と commit-msg が成功した。

### 全体検証

- `bash scripts/check-app.sh` が終了コード0で成功した。gofmt・vet・全通常テスト、lint 0 issues、Git 非依存3条件、race 付き通常／結合テストを実行した。結合テストだけの成功とは扱わない。
- 同じ本番 image ID `sha256:9f746fce41812fbbac516856b39839cee38cb13d5446dd55e96cbd5e45509dbd` で HTTP/MCP、ローカル／公開 Host・Origin、PORT、不正設定の非0終了、SIGTERM/SIGINT を確認した。
- PR 用本文の規約検査と、変更5文書のローカルリンク103件の参照先確認（アンカーは対象外）が成功した。記録コミットのフックと最終の範囲・差分・本文の確認結果は Issue #92 を参照する。

### AI のセルフレビューと未実施項目

評価種別を先に文字列化し、未知値ではレビューの構築や JSON 化へ進まない順序を維持した。レビューは新しい配列へ取得順に変換し、入力の配列や条件を書き換えない。ID は10進文字列、AppID と時刻は元の数値型、未指定の条件は新たに正規化しない。空の配列を nil にしていない。人によるレビューは未実施。

GitHub 認証・Docker・必須フックは権限付きで実行した。既存の一時 Docker 設定を使用し、検査対象・設定・フックを省略・緩和していない。

push 前のため pre-push、送信済み head/base の PR 作成検査、コード CI は未実施。実 Steam／LLM 接続・Render 配置、coverage 閾値・分岐・mutation（Stryker を含む）は未導入／未実施であり、今回の結果を品質目標の100%達成とは扱わない。

## 参照仕様・ADR

- [データ・調査仕様](../specs/data-and-research.md) のレビューと収集条件、上限・部分失敗
- [収集 CLI と保存先の仕様](../specs/review-cli.md) の JSON 変換と writer、Docker と検証
- [ADR-0006](../adr/0006-collection-output-boundary.md) の JSON 出力データ構築と writer の境界、[ADR-0012](../adr/0012-steam-domain-types.md) の数値と JSON の変換、[ADR-0013](../adr/0013-domain-folder-layers.md) の領域内の adapter の責務
- [操作一覧](../use-cases.md) の純粋な JSON 変換と MCP の再利用、[開発手順](../development.md) の検証・必須フック・PR 作成手順

## 関連 Issue

Closes #92
