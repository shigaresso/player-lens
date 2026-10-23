# ADR-0012：Steam の ID・時刻・列挙値を名前付き数値型で扱う

## 状態と対象

採用。[Issue #43](https://github.com/shigaresso/player-lens/issues/43) の方針を記録する。実装・検証の状態は同 Issue と作業記録で区別する。Steam HTTP adapter、収集、保存、CLI、MCP に適用する。

## 背景・比較

組み込み型だけでは App ID と時刻、異なる API 列挙値の取り違えを検出できない。recommendationid は公式の uint64 と異なり string と独自の正数検査で保持していた。

- 型エイリアスは元の型と区別できないため採用しない。
- `AppID uint32`、`RecommendationID uint64`、`TimestampCreated uint32`、`TimestampUpdated uint32` を独立した定義型として採用する。これらの名前は API の項目名に基づくアプリ側の名前であり、公式資料の型名そのものではない。
- API の3列挙値は公式名の `EUserReviewsReviewType`・`EUserReviewsAppReviewsFilter`・`EUserReviewsPurchaseType` を int32 に基づく定義型にする。
- 文字列・bool の項目とアプリ独自の件数・利用制限には型を追加しない。

## 決定と境界

Steam の応答は専用の受信データで読み、recommendationid の文字列を10進の uint64 に変換して領域へ渡す。省略・null・空・不正構文・範囲超過は不正応答とする。0 は uint64 の範囲内として受け付け、独自の正数条件は適用しない。先頭ゼロは数値に正規化し、収集では同じ数値の ID を一度だけ保持する。ID の実在を数値変換で保証するものではない。

AppID は既存の正数という取得対象の条件を維持し、時刻は UNIX 秒・uint32 の範囲を維持する。名前付き型の導入自体で、値の実在や呼び出し後の変更の有効性を保証したとは扱わない。

既存の `ReviewType` はユーザー向けの all/positive/negative を表す文字列の契約とし、HTTP adapter が `EUserReviewsReviewType` に変換する。API の数値コードを CLI・MCP の文字列入力へ漏らさない。

出力境界の `jsonoutput` は schema_version=1 と既存の項目名を維持し、数値の RecommendationID を10進の JSON 文字列に変換する。0 と最大値も精度損失なく出力する。JSON のレビュー ID は文字列のままだが、先頭ゼロという元の表記は保持しない。CLI と MCP は同じ変換を使う。

領域の数値型へ JSON の読み書きや I/O を組み込まない。受信・出力の表現と領域の型の変換は各境界が担当する。新しいパッケージ、interface、依存ライブラリ、再試行を導入しない。

## 利点と制約

Steam の項目と型の対応を名前で追え、ID・時刻・列挙値の混同をコンパイラで検出できる。Go の呼び出し元は新しい型・フィールド名への更新が必要となる。境界での変換と、定義型のゼロ値を含む検証は引き続き必要である。

## 参照

- [Steam GetAppReviews と列挙値](https://partner.steamgames.com/doc/webapi/IUserReviewsService#GetAppReviews)
- [strconv.ParseUint](https://pkg.go.dev/strconv#ParseUint)
- [1ページ取得仕様](../specs/steam-review-page.md)、[収集仕様](../specs/collect-reviews.md)、[CLI・JSON 仕様](../specs/review-cli.md)、[MCP 仕様](../specs/review-mcp.md)
- ADR-0004・0005・0006・0007
