# アプリケーションのユースケース

## 導入済みの操作

| 操作 | 目的と責務 | 入力と依存 | 結果と失敗 | 呼び出し元の状態 |
| --- | --- | --- | --- | --- |
| `steam.CollectReviews` | レビューを上限付きで収集し、カーソル・重複・終了条件を管理する | context、PageFetcher、App ID、CollectionOptions | Collection と終了理由。不正入力はゼロ値、途中失敗は部分結果と原因を保持したエラー | ユニットテストと手動確認から呼び出した。CLI・Web・MCP は未接続 |

[収集仕様](specs/collect-reviews.md) と [ADR-0005](adr/0005-bounded-review-collection.md) に契約と判断を記載する。

## パーツとの境界

`steam.Client.FetchReviewsPage` は1回の HTTP 通信と入力・応答の検査、応答 Body の解放を担当する adapter とする。収集操作とは区別し、ページングや収集全体の件数・時間・部分失敗を重複させない。

`NormalizePageOptions` と `NormalizeCollectionOptions` は通信しない取得条件の正規化と検査を担当する。保存・分析・レポート生成は未導入とする。

入口側が HTTP クライアントと `Client` を組み立て、同じ `CollectReviews` へ渡す。操作内に具体的な HTTP クライアントや DB を固定せず、同じ Go アプリケーション内の呼び出しに HTTP・MCP 通信を挟まない。
