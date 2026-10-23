# ADR-0004：公開 IUserReviewsService を使う

## 状態と対象

採用。[Issue #17](https://github.com/shigaresso/player-lens/issues/17) の Steam レビュー HTTP adapter に適用する。呼び出し契約は [1ページ取得仕様](../specs/steam-review-page.md) を参照する。

## 背景・比較・決定

旧 Store `/appreviews/{appid}` は公式に非推奨となった。現行の `https://api.steampowered.com/IUserReviewsService/GetAppReviews/v1/` を使い、引数は `input_json` へまとめる。公開取得にキーは不要なので、秘密情報を増やさず標準ライブラリだけで実装する。

クライアント内で再試行せず、将来の収集操作が回数・再開・部分失敗を統括する。リダイレクトを追わず、1回10秒、2 MiBを本アプリの上限とする。1ページ100件は Steam の仕様に対応する。いずれも Steam の QPS 制限を推定した数字ではない。

取得条件は通常の `PageOptions` を維持し、既定値の適用と検査を公開関数 `NormalizePageOptions` へ集約する。非公開の補助関数と比較し、別パッケージで検査だけを利用できることを優先した。検査済みの専用型を増やす設計は現段階では採用せず、取得処理も必ず共通検査を通す。戻り値の変更後の有効性を型で保証するものではない。

## 評価種別の拡張（Issue #33）

取得後に全評価から間引く方式では、通信量が増え、指定件数の意味が変わる。Steam API の review_type を使い、公開する文字列 all / positive / negative と API の数値0 / 1 / 2を分離する。Go は文字列を基底型とする ReviewType と名前付き定数を使い、PageOptions の正規化を CollectionOptions・CLI・MCP で共通利用する。型だけでは未知値を防げないため、通信前に必ず検査する。

既定は全評価を維持し、JSON の options.review_type は schema_version=1 の追加項目とする。収集・JSON 出力・MCP エラー結果でも条件を引き継ぐ。並び順・期間などの別条件は今回拡張しない。

## 名前付き数値型（Issue #43）

ID・時刻と API の3列挙値の型、recommendationid の0の受理と数値化、外部 JSON との変換は [ADR-0012](0012-steam-domain-types.md) を採用する。外部の文字列入力・出力と内部の数値列挙型の変換は CLI・MCP・JSON の境界に集約する。Issue #33 の文字列型を内部へ伝播する方針をここで置き換える。

## 利点と制約

現行仕様に合わせて取得条件と結果を明示できる。キーなしの応答は最大10分キャッシュされ、共有の制限で429になり得る。全作品・全言語の成功や常に最新の応答を保証しない。実 API の1件の読取で `response` wrapper と文字列 ID を確認したが、通常テスト・CI は制御したローカル通信で契約を検証する。

## 公式資料

- [GetAppReviews](https://partner.steamgames.com/doc/webapi/IUserReviewsService#GetAppReviews)
- [旧 API からの移行](https://partner.steamgames.com/doc/webapi/IUserReviewsService#migrating)
- [Service Interfaces と input_json](https://partner.steamgames.com/doc/webapi_overview#Service_Interfaces)
- [言語コード](https://partner.steamgames.com/doc/store/localization/languages)
