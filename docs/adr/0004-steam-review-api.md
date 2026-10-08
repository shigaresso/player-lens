# ADR-0004：公開 IUserReviewsService を使う

## 状態と対象

採用。[Issue #17](https://github.com/shigaresso/player-lens/issues/17) の Steam レビュー HTTP adapter に適用する。呼び出し契約は [1ページ取得仕様](../specs/steam-review-page.md) を参照する。

## 背景・比較・決定

旧 Store `/appreviews/{appid}` は公式に非推奨となった。現行の `https://api.steampowered.com/IUserReviewsService/GetAppReviews/v1/` を使い、引数は `input_json` へまとめる。公開取得にキーは不要なので、秘密情報を増やさず標準ライブラリだけで実装する。

クライアント内で再試行せず、将来の収集操作が回数・再開・部分失敗を統括する。リダイレクトを追わず、1回10秒、2 MiB、100件を上限とする。これらは本アプリの制限であり、Steam の QPS 制限を推定した数字ではない。

## 利点と制約

現行仕様に合わせて取得条件と結果を明示できる。キーなしの応答は最大10分キャッシュされ、共有の制限で429になり得る。全作品・全言語の成功や常に最新の応答を保証しない。実 API の1件の読取で `response` wrapper と文字列 ID を確認したが、通常テスト・CI は制御したローカル通信で契約を検証する。

## 公式資料

- [GetAppReviews](https://partner.steamgames.com/doc/webapi/IUserReviewsService#GetAppReviews)
- [旧 API からの移行](https://partner.steamgames.com/doc/webapi/IUserReviewsService#migrating)
- [Service Interfaces と input_json](https://partner.steamgames.com/doc/webapi_overview#Service_Interfaces)
- [言語コード](https://partner.steamgames.com/doc/store/localization/languages)
