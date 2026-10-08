# アプリケーションのユースケース

## 導入済みの操作

| 操作 | 目的と責務 | 入力と依存 | 結果と失敗 | 呼び出し元の状態 |
| --- | --- | --- | --- | --- |
| `steam.CollectReviews` | レビューを上限付きで収集し、カーソル・重複・終了条件を管理する | context、PageFetcher、App ID、CollectionOptions | Collection と終了理由。不正入力はゼロ値、途中失敗は部分結果と原因を保持したエラー | 収集・保存操作、テストと手動確認から呼び出した |
| `steam.CollectAndSaveReviews` | 収集と結果保存の順序、部分結果・複合失敗・保存期限を管理する | context、PageFetcher、CollectionSaver、App ID、CollectionOptions | Collection と収集/保存の原因を保持したエラー。保存は一度だけ行う | CLI から接続した。Web・MCP は未接続 |

[収集仕様](specs/collect-reviews.md) と [ADR-0005](adr/0005-bounded-review-collection.md) に契約と判断を記載する。

## パーツとの境界

`steam.Client.FetchReviewsPage` は1回の HTTP 通信と入力・応答の検査、応答 Body の解放を担当する adapter とする。収集操作とは区別し、ページングや収集全体の件数・時間・部分失敗を重複させない。

`NormalizePageOptions` と `NormalizeCollectionOptions` は通信しない取得条件の正規化と検査を担当する。`jsonoutput.MarshalCollection` は通信しない JSON 変換、`jsonoutput.Saver` は writer への出力を担当する。DB の永続保存・分析・レポート生成は未導入とする。

入口側が HTTP クライアントと `Client` を組み立て、同じ `CollectReviews` へ渡す。操作内に具体的な HTTP クライアントや DB を固定せず、同じ Go アプリケーション内の呼び出しに HTTP・MCP 通信を挟まない。

CLI の `reviewcli.Run` は引数の解析と `CollectAndSaveReviews` への接続を担当し、収集や保存の手順を重複させない。main が JSON Saver を渡す。保存先を変更するときは同じ CollectionSaver 契約を実装して組み立て箇所で選び、操作と入力解析を共通で使う。[CLI 仕様](specs/review-cli.md) と [ADR-0006](adr/0006-collection-output-boundary.md) に責務を記載する。
