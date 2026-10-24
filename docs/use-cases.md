# アプリケーションのユースケース

## 導入済みの操作

| 操作 | 目的と責務 | 入力と依存 | 結果と失敗 | 呼び出し元の状態 |
| --- | --- | --- | --- | --- |
| `steam.CollectReviews` | レビューを上限付きで収集し、カーソル・重複・終了条件を管理する | context、PageFetcher、App ID、CollectionOptions | Collection と終了理由。不正入力はゼロ値、途中失敗は部分結果と原因を保持したエラー | 収集・保存操作、MCP、テストと手動確認から呼び出した |
| `steam.CollectAndSaveReviews` | 収集と結果保存の順序、部分結果・複合失敗・保存期限を管理する | context、PageFetcher、CollectionSaver、App ID、CollectionOptions | Collection と収集/保存の原因を保持したエラー。保存は一度だけ行う | CLI から接続した。MCP は保存しないためこの操作を使わない。Web は未接続 |

[収集仕様](specs/collect-reviews.md) と [ADR-0005](adr/0005-bounded-review-collection.md) に契約と判断を記載する。

## パーツとの境界

`steam.Client.FetchReviewsPage` は1回の HTTP 通信と入力・応答の検査、応答 Body の解放を担当する adapter とする。収集操作とは区別し、ページングや収集全体の件数・時間・部分失敗を重複させない。

Issue #38 で adapter 内のリクエスト変換を分離した。Issue #43 では文字列の評価種別から数値への変換を入力境界へ移した。`newReviewPageRequest`、`encodeReviewPageQuery` は、正規化済みの条件を Steam API の入力とクエリへ変換する通信なしの処理であり、新しいユースケースではない。

Issue #39 で応答解釈も分離した。`decodeReviewPage` は本文と取得条件からページを作り、Issue #43 の `parseRecommendationID` は ID の文字列を0を含む uint64 に変換する。Body の読み取り・サイズ制限・解放は `FetchReviewsPage` に残す。

`NormalizePageOptions` と `NormalizeCollectionOptions` は通信しない取得条件の正規化と検査を担当する。`jsonoutput.MarshalCollection` は通信しない JSON 変換、`jsonoutput.Saver` は writer への出力を担当する。DB の永続保存・分析・レポート生成は未導入とする。

入口側が HTTP クライアントと `Client` を組み立て、同じ `CollectReviews` へ渡す。操作内に具体的な HTTP クライアントや DB を固定せず、同じ Go アプリケーション内の呼び出しに HTTP・MCP 通信を挟まない。

CLI の `reviewcli.Run` は引数の解析と `CollectAndSaveReviews` への接続を担当し、収集や保存の手順を重複させない。main が JSON Saver を渡す。保存先を変更するときは同じ CollectionSaver 契約を実装して組み立て箇所で選び、操作と入力解析を共通で使う。[CLI 仕様](specs/review-cli.md) と [ADR-0006](adr/0006-collection-output-boundary.md) に責務を記載する。

MCP の `reviewmcp.NewHandler` は `collect_reviews` の入力と HTTP 制限、実行期限、Collection の応答変換を担当する。`CollectReviews` を直接呼び、JSON の純粋変換だけを再利用する。保存操作は呼ばない。分析は接続元の ChatGPT が担当し、バックエンドの LLM 呼び出しは未導入とする。[MCP 仕様](specs/review-mcp.md) と [ADR-0007](adr/0007-review-mcp.md) に責務を記載する。

MCP の開始回数と同時数は handler が所有する入口の制限とし、収集操作・CLI へ混ぜない。[利用制限仕様](specs/mcp-usage-limits.md) と [ADR-0008](adr/0008-mcp-usage-limits.md) に従う。

Issue #35 の `analysis_guide` は MCP の案内と結果変換の責務であり、新たな分析ユースケースではない。共通本文をバイナリへ埋め込み、収集結果とともに提供する。steam・CLI・保存先へ分析方針を混ぜず、分析は接続元の AI、採否は人間に残す。[レビュー分析仕様](specs/review-analysis.md) に従う。

公開 MCP の `mcpaccess.Access` は公開 URL の検査と HTTP の許可 Host・cross-origin 保護だけを担当する。main が組み立て、`reviewmcp.NewHandler` へ渡す。未指定はローカル専用、指定時は匿名公開とし、認証・JWT・OAuth metadata は提供しない。収集ユースケース・CLI へ公開設定を混ぜない。[公開 MCP 仕様](specs/mcp-public-access.md) と [ADR-0009](adr/0009-public-mcp-access.md) に従う。

## 評価種別（Issue #33）

Issue #43 では `PageOptions` と `CollectionOptions` の `ReviewType` は独立した数値列挙型 ReviewType の0 / 1 / 2で表す。値は公式の定数から定義し、HTTP adapter で EUserReviewsReviewType に変換する。文字列は CLI・MCP の入力境界で ParseReviewType により解析し、JSON 出力境界の FormatReviewType で戻す。未知の数値は共通正規化と送信・出力で拒否する。収集は全ページへ条件を渡し、Collection と JSON の options.review_type に保持する。CLI の --review-type と MCP の review_type は同じ操作へ接続し、独自のページングや取得後の間引きを追加しない。保存先の interface とバックエンド分析の未導入状態は維持する。

## Steam の数値型（Issue #43）

AppID・RecommendationID・TimestampCreated・TimestampUpdated は steam の定義型を各操作と呼び出し元へ引き継ぐ。評価種別の列挙型は内部全体で引き継ぎ、残る2列挙型は HTTP adapter の数値表現に使う。領域の型に JSON の読み書きを持たせず、HTTP の受信と jsonoutput の出力で変換する。選択理由と外部形式は [ADR-0012](adr/0012-steam-domain-types.md) に従う。
