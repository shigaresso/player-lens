# Steam レビューの1ページ取得

## 範囲

[Issue #17](https://github.com/shigaresso/player-lens/issues/17) の HTTP adapter の契約を定める。複数ページの操作は [収集仕様](collect-reviews.md) と [ユースケース](../use-cases.md) を参照する。この adapter に保存、取得時刻・本文の版、再試行、MCP の責務は追加しない。

## API と入力

[ADR-0004](../adr/0004-steam-review-api.md) の公開 API を GET で呼ぶ。キーは使わず、`format=json` と URL エンコードした `input_json` を送る。`appid` は正の uint32、`filter=1`（Recent）、`review_type` は条件に従い0（All）・1（Positive）・2（Negative）、`purchase_type=1`（All）、`filter_offtopic_activity=true` とする。

Issue #33 で評価種別を追加し、Issue #43 で内部を `ReviewType` の数値列挙値にした。CLI・MCP 境界では空と `all` を0、`positive` を1、`negative` を2に変換する。大文字・空白・未知値は通信前に拒否する。API の数値への変換は HTTP adapter が担当し、取得後のローカルな間引きで代用しない。

`PageOptions` の言語は既定 `english`、カーソルは `*`、件数は20とする。件数は1〜100、言語は小文字の英字で始まる英小文字・数字・ハイフン・アンダースコア32文字以内、カーソルは4096バイト以内とする。言語コードの対応は Steam 側に依存し、この構文検査だけで対応言語を保証しない。不正入力は通信前に拒否する。

先頭ページを表す `*` は公開定数 `FirstPageCursor` とし、既定値の適用と呼び出し側で使う。続きのページには API が返した次のカーソルをそのまま渡す。

`NormalizePageOptions` を公開し、他のパッケージからも通信なしで既定値の適用と同じ検査を使えるようにする。入力を変更せず、成功時は正規化したコピー、不正時は `PageOptions{}` と `ErrInvalidArgument` を返す。正規化済みの入力への再適用では値を変えない。戻り値は通常の `PageOptions` なので、書き換え後の有効性を型で保証するものではない。

Steam の1ページ最大100件は公開定数 `MaxPageSize` で表す。`FetchReviewsPage` は必ず `NormalizePageOptions` を通してから通信し、呼び出し側が事前検査していなくても不正な取得条件を拒否する。AppID の検査は引き続き `FetchReviewsPage` の責務とする。

HTTP クライアントと接続先は外から渡す。空の接続先は公式 URL を使い、指定先は絶対 http/https URL とし、userinfo・query・fragment を禁止する。接続プールの終了は呼び出し側の責務とする。

### リクエスト変換の境界（Issue #38）

`FetchReviewsPage` は必ず `NormalizePageOptions` を通し、未知の評価種別を通信前に `ErrInvalidArgument` で拒否する。Issue #64 では、その成功後に呼ぶ `review_page_request.go` の `newReviewPageRequest` から同じ検査を除いた。正の App ID と正規化済みの条件を渡すことを呼び出し側の契約とし、内部の ReviewType を EUserReviewsReviewType へ変換して固定の API 条件を設定する。変換だけを行うため error 戻り値は持たない。`encodeReviewPageQuery` は JSON とクエリへ変換する。これらは通信・時刻・保存に依存しない。既定値と入力検査は `NormalizePageOptions`、期限と送信は `FetchReviewsPage`、応答処理と Body の Close は `readReviewPageResponse` が担当する。

API 側の数値定数は [公式の列挙名](https://partner.steamgames.com/doc/webapi/IUserReviewsService#Enums) と同じ `k_EUserReviewsReviewType_All`・`k_EUserReviewsReviewType_Positive`・`k_EUserReviewsReviewType_Negative`、`k_EUserReviewsAppReviewsFilter_Recent`、`k_EUserReviewsPurchaseType_All` を使う。これらは非公開とし、API へ送る値を明示的な数値で定義する。

## 結果と失敗

成功時は AppID、既定値適用後の条件、レビュー配列、次のカーソルを返す。recommendationid は0を含む uint64 に基づく `RecommendationID` として保持し、本文、言語、推薦、作成・更新の UNIX 秒を返す。未知のフィールドは無視する。空配列は成功で、空ページのカーソルは省略可能とする。

HTTP 200 以外はステータスを持つエラーとし、リダイレクトを追わず、再試行しない。JSON 不正、`response` 欠落、不正な ID、要求件数を超える配列、非空配列でのカーソル欠落は不正応答とする。失敗時は部分結果を返さずゼロ値のページを返す。

呼び出し側の context と期限を引き継ぎ、1回を最大10秒に制限する。応答は最大2 MiBとする。応答 Body は成功・失敗とも adapter が閉じ、読込・Close・通信・context の原因は `errors.Is` / `errors.As` で追跡可能にする。複数原因は併せて返す。

### 応答解釈の境界（Issue #39）

`review_page_response.go` の `decodeReviewPage` が本文の JSON 解析、wrapper・件数・カーソルの検査、空配列の正規化とページ生成を担当する。Issue #43 では専用の受信データから `parseRecommendationID` で ID を10進の uint64 に変換する。省略・null・空・構文不正・範囲超過は拒否し、0は受け付ける。先頭ゼロは数値に正規化する。Issue #68 で、受信型 reviewPageReview から1件の内部 Review を作る処理を非公開の convertReview に分離した。ID の数値変換と各項目のコピー、不正時の Review のゼロ値と ErrInvalidResponse・数値変換原因は convertReview が担当する。decodeReviewPage は JSON とページ全体を検査して変換を順に呼び、1件でも失敗した場合は先行レビューも含めて返さず、ページの完全なゼロ値と原因を返す。これらは通信しない。Issue #66 では、HTTP ステータス、Body の読み取り・2 MiB 制限、解釈の呼び出し、Close と複合エラーを `read_review_page_response.go` の `readReviewPageResponse` にまとめた。Do が成功して返した有効な応答と、正の App ID・正規化済みの条件を受け取り、Body を読み取りから Close まで所有する。解釈を終えてから Close し、Close 失敗ではページをゼロ値にして元の原因と併せて返す。読み取り中の同じ goroutine の panic も Close した後に呼び出し側へ伝える。FetchReviewsPage は入力・要求構築・送信と取得用 context を所有し、応答処理が Close を完了してから context を解放する。失敗時のゼロ値と原因保持を維持する。

## 検証

応答解釈と ID 検査は、同じパッケージのユニットテストから非公開関数を直接呼び、HTTP を使わず検証する。ID の構文・数値境界、ページの生成・空配列・未知フィールド、不正応答でのゼロ値、JSON の構文・型エラーの原因保持を確認する。

HTTP adapter の通常テストでは `httptest` とテスト用 Transport / Body を使い、実 Steam API は使わない。正常・空ページ、入力境界、HTTP エラーと非再試行、JSON・スキーマ・サイズ、期限・キャンセル、Body の解放とエラーを確認する。作業記録は [Issue #17 の記録](../worklogs/17-steam-review-page.md) と [Issue #39 の記録](../worklogs/39-steam-response-decoding.md) に残す。API を呼べることと、収集・保存・分析が実装済みであることを区別する。 Issue #64 の検査と変換の整理は [検証記録](../worklogs/64-steam-request-validation.md) に残す。 Issue #66 の応答処理と Body の所有の整理は [検証記録](../worklogs/66-steam-response-lifecycle.md) に残す。

## Steam の名前付き数値型（Issue #43）

[ADR-0012](../adr/0012-steam-domain-types.md) に従い、AppID・RecommendationID・TimestampCreated・TimestampUpdated を領域から各呼び出し元へ渡す。API の列挙値は公式名の int32 型で表し、評価種別は内部で独立した ReviewType を使い、各値を公式の定数から定義する。HTTP adapter で EUserReviewsReviewType に変換する。文字列との変換は CLI・MCP の入力と JSON の出力境界だけが担当する。応答データの文字列 ID と領域の数値型を分け、数値変換の原因も ErrInvalidResponse とともに保持する。
