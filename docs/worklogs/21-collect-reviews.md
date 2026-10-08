# Issue #21：Steam レビューの上限付き収集

## 範囲と参照

[Issue #21](https://github.com/shigaresso/player-lens/issues/21) を origin/main `74d1b5b51d791b2839a57a4f717838dc8dd340c4` から `codex/21-collect-steam-reviews` で進めた。ユーザーの指示に従い push・PR 作成・Issue クローズは行わない。

文書一覧、データ・調査仕様のレビューと収集条件・ジョブと上限・部分失敗、1ページ取得仕様、公開デモ仕様の実装状態、開発手順、採用済み ADR-0001・0002・0003・0004 を参照した。契約を [収集仕様](../specs/collect-reviews.md)、判断を [ADR-0005](../adr/0005-bounded-review-collection.md)、操作の境界を [ユースケース](../use-cases.md) に記録した。

## 契約単位の Red・Green

関連テストは固定版 Compose で `go test -count=1 ./internal/steam` を実行し、Red では対象関数を `-run` で選んだ。同じ契約の入力違い・境界値は名前付きテーブルにまとめた。

| 契約 | Red | Green とコミット |
| --- | --- | --- |
| 収集条件の正規化 | CollectionOptions・公開関数・上限定数が未定義でコンパイル失敗 | 既定値・境界・不正値・コピーと再適用が成功した。63431a4 |
| 正常な複数ページ収集 | CollectReviews と結果・終了理由が未定義でコンパイル失敗 | カーソル・残件数・空ページ・件数/ページ上限と重複の最初の値・順序を確認した。00be82f |
| 途中失敗の部分結果 | 失敗時にゼロ値を返し、条件・取得済みレビュー・ページ数を失う失敗 | 部分結果と StopFailed を返し、429・通信・context・複数の原因を保持した。46109f5 |
| カーソル循環 | 循環を通常のページ上限として成功扱いする失敗 | 循環の有限な終了と空ページ・件数上限・循環・ページ上限の優先順位を確認した。11b862d |
| context の期限と解放 | 2分の期限がない/長い期限を維持し、派生 context を解放せず、終了済み・キャンセル後に取得する失敗 | 短い期限・値の保持、最大2分、取得前の終了検査と派生 context の解放を確認した。e0718ac |

実装後の入口の拒否、最大1万件、errors.Is による循環の識別、失敗時の解放と実 Client のローカル接続は既に Green だった。回帰テストのみの `f000891` にまとめ、人工的な Red は作らなかった。HTTP テストの記録は mutex で同期し、新しい回帰テストを `go test -race -count=1 ./internal/steam -run 'TestCollectReviews(InvalidInput|MaximumReviews|FailureReleasesContext|WithHTTPClient)'` でも確認した。

## 全体検証

`bash scripts/check-app.sh` の整形・vet・通常テスト、golangci-lint 0件、通常テストを含む `go test -tags=integration -race -count=1 ./...`、本番用イメージ検査の完了を確認した。

本番イメージ ID `sha256:74fc2ec8fcaeee2f910e45035718303e606aa16a6b3ab924a45781c29665e3d5` で PORT 8080・10000、HTTP 応答、SIGTERM・SIGINT の正常終了、不正 PORT の非0終了を確認した。サーバーの入口は変更していないため、バイナリは変更前と同じとなった。Go パッケージは通常・race テストと手動の実行プログラムで検証した。

全体検査後の結果表示用シェルが zsh の予約変数 `status` と衝突して終了コード1となった。検査本体の完了ログ（両パッケージの通常/race の成功、lint 0件、本番用イメージの最後の検査）と終了処理を確認し、表示処理の失敗と区別した。同じ検査を重ねて再実行しなかった。

Docker は公開イメージ用の一時的な DOCKER_CONFIG を使い、認証ヘルパーの待機を避けた。普段の Docker 設定を変更せず、必須フックを回避しなかった。各コミットの pre-commit は共通144テスト・actionlint・シェル構文・空白と Go の index の整形、commit-msg は日本語の本文付きメッセージを検査した。

## 実 Steam API の手動確認

2026-10-08 に1回だけ、AppID 440・english・PageSize 2・MaxReviews 3・MaxPages 2 で CollectReviews と Client を実行した。

```text
AppID=440 reviews=3 pages=2 requests=2 reason=review_limit error=<nil>
```

実 API への2リクエストでカーソルを引き継ぎ、残件数を取得して上限で終了した。本文は出力・保存せず、件数・成功ページ数・呼び出し数・終了理由だけを記録した。実行プログラムは一時ディレクトリを開発用イメージへマウントし、リポジトリと CI へ追加しなかった。手動確認は一度の少量取得であり、全作品・全言語・常に最新の応答を保証しない。

通常テストはテスト用 PageFetcher と httptest のローカルサーバーを使う。1万件のテストも生成した応答だけで行い、Steam に接続しない。

## セルフレビューと未導入項目

AI のセルフレビューで入力の共通検査、終端の優先順位、最大件数・ページ数・期限、部分結果、最初の ID の保持、原因と context の解放、HTTP adapter の既存の責務と秘密情報・外部本文が差分に入らないことを確認した。

CLI・Web・MCP の入口、保存・本文版・ジョブ・再開、再試行・並列取得、coverage・分岐・mutation の検査、Render 配置、人によるレビュー、このブランチの GitHub CI は未導入・未実施・未確認として区別する。
