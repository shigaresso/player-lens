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

評価種別は既存の `ReviewType` と3定数の構成を内部で再利用し、`type ReviewType EUserReviewsReviewType` による独立した数値列挙型にする。ReviewTypeAll・ReviewTypePositive・ReviewTypeNegative は公式名の k_EUserReviewsReviewType_* を ReviewType に変換して定義する。公式型の別名定数ではなく内部の取得条件の値として使い、HTTP adapter で EUserReviewsReviewType に明示的に変換する。CLI・MCP の文字列は ParseReviewType で解析し、JSON は FormatReviewType で文字列に戻す。未知の整数は正規化・送信・出力で拒否する。ゼロ値は全評価なので、空の結果にも options.review_type=all を明示する。

出力境界の `jsonoutput` は schema_version=1 と既存の項目名を維持し、数値の RecommendationID を10進の JSON 文字列に変換する。0 と最大値も精度損失なく出力する。JSON のレビュー ID は文字列のままだが、先頭ゼロという元の表記は保持しない。CLI と MCP は同じ変換を使う。

領域の数値型へ JSON の読み書きや I/O を組み込まない。受信・出力の表現と領域の型の変換は各境界が担当する。新しいパッケージ、interface、依存ライブラリ、再試行を導入しない。

## HTTP リクエストの検査と変換（Issue #64）

未知の評価種別を拒否する責務は、公開の FetchReviewsPage が必ず呼ぶ NormalizePageOptions に集約する。非公開の newReviewPageRequest は正の App ID と正規化済みの条件を受け取り、公式の API 型への変換と固定条件の設定を担当する。同じ評価種別の検査と error 戻り値を除き、呼び出し元の到達しない分岐も除く。

内部変換にも同じ検査を残す案と比較し、公開入口が検査した値だけを渡す契約を明示する方式を採用する。PageOptions の専用型や公開 API は増やさず、ADR-0004 の通常の入力型と送信前の共通検査を維持する。元の private 拒否テストの入力は公開入口の非通信・ゼロ値・原因識別へ引き継ぎ、3評価種別の変換は内部と HTTP のテストで確認する。

## 利点と制約

Steam の項目と型の対応を名前で追え、ID・時刻・列挙値の混同をコンパイラで検出できる。Go の呼び出し元は新しい型・フィールド名への更新が必要となる。境界での変換と、定義型のゼロ値を含む検証は引き続き必要である。

## 参照

- [Steam GetAppReviews と列挙値](https://partner.steamgames.com/doc/webapi/IUserReviewsService#GetAppReviews)
- [strconv.ParseUint](https://pkg.go.dev/strconv#ParseUint)
- [1ページ取得仕様](../specs/steam-review-page.md)、[収集仕様](../specs/collect-reviews.md)、[CLI・JSON 仕様](../specs/review-cli.md)、[MCP 仕様](../specs/review-mcp.md)
- ADR-0004・0005・0006・0007
