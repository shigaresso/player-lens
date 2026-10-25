# アプリケーションのユースケース

既存の領域フォルダを維持し、その内部を domain・application・adapter に分類した。[ADR-0013](adr/0013-domain-folder-layers.md) に配置と依存を記録する。以下の steamdomain・steamapplication・steamadapter は internal/steam 配下の各レイヤー、mcpdomain・mcpapplication は internal/reviewmcp 配下の各レイヤーを指す。reviewmcp・reviewcli・jsonoutput は各領域の adapter を指す。

## 導入済みの操作

| 操作 | 目的と責務 | 入力と依存 | 結果と失敗 | 呼び出し元の状態 |
| --- | --- | --- | --- | --- |
| `steamapplication.CollectReviews` | レビューを上限付きで収集し、カーソル・重複・終了条件を管理する | context、PageFetcher、App ID、CollectionOptions | Collection と終了理由。不正入力はゼロ値、途中失敗は部分結果と原因を保持したエラー | 収集・保存操作、MCP、テストと手動確認から呼び出した |
| `steamapplication.CollectAndSaveReviews` | 収集と結果保存の順序、部分結果・複合失敗・保存期限を管理する | context、PageFetcher、CollectionSaver、App ID、CollectionOptions | Collection と収集/保存の原因を保持したエラー。保存は一度だけ行う | CLI から接続した。MCP は保存しないためこの操作を使わない。Web は未接続 |
| `mcpapplication.CollectReviews` | MCP の開始回数・同時枠と8秒の期限を管理し、共通の収集操作を実行する | context、PageFetcher、解析済みの App ID と CollectionOptions、共有 Limiter | Result に収集結果・拒否理由・元の失敗を分けて保持する。枠は終了時に返却する | MCP adapter が入力検査後に呼び出す |

[収集仕様](specs/collect-reviews.md) と [ADR-0005](adr/0005-bounded-review-collection.md) に契約と判断を記載する。

## パーツとの境界

`steamadapter.Client.FetchReviewsPage` は1回の HTTP 通信と入力・応答の検査、応答 Body の解放を担当する adapter とする。収集操作とは区別し、ページングや収集全体の件数・時間・部分失敗を重複させない。

Issue #38 で adapter 内のリクエスト変換を分離した。Issue #43 では文字列の評価種別から数値への変換を入力境界へ移した。`newReviewPageRequest`、`encodeReviewPageQuery` は、正規化済みの条件を Steam API の入力とクエリへ変換する通信なしの処理であり、新しいユースケースではない。

Issue #39 で応答解釈も分離した。`decodeReviewPage` は本文と取得条件からページを作り、Issue #43 の `parseRecommendationID` は ID の文字列を0を含む uint64 に変換する。Body の読み取り・サイズ制限・解放は `FetchReviewsPage` に残す。

Issue #58 で、重複・カーソル・成功ページ数・終了判断を steam/domain の CollectionState にまとめた。生成時に App ID と条件を検査・正規化し、NextPage と AddPage で続行と状態更新を扱う。CollectReviews は取得・期限・キャンセルと失敗の原因、Fail による失敗確定を担当する。Result はレビュー配列もコピーして返す。状態は通信や context を持たず、新しいユースケースではない。[ADR-0014](adr/0014-steam-collection-state.md) に境界を記録する。

`NormalizePageOptions` と `NormalizeCollectionOptions` は通信しない取得条件の正規化と検査を担当する。`jsonoutput.MarshalCollection` は通信しない JSON 変換、`jsonoutput.Saver` は writer への出力を担当する。DB の永続保存・分析・レポート生成は未導入とする。

入口側が HTTP クライアントと `Client` を組み立て、同じ `CollectReviews` へ渡す。操作内に具体的な HTTP クライアントや DB を固定せず、同じ Go アプリケーション内の呼び出しに HTTP・MCP 通信を挟まない。

CLI の `reviewcli.Run` は引数の解析と `CollectAndSaveReviews` への接続を担当し、収集や保存の手順を重複させない。main が JSON Saver を渡す。保存先を変更するときは同じ CollectionSaver 契約を実装して組み立て箇所で選び、操作と入力解析を共通で使う。[CLI 仕様](specs/review-cli.md) と [ADR-0006](adr/0006-collection-output-boundary.md) に責務を記載する。

MCP の `reviewmcp.NewHandler` は HTTP/SDK と共有する制限状態を組み立てる。adapter は入力・HTTP 制限と Collection の応答変換、mcpapplication.CollectReviews は利用枠と実行期限を担当する。application から steamapplication.CollectReviews を呼び、adapter は JSON の純粋変換だけを再利用する。保存操作は呼ばない。分析は接続元の ChatGPT が担当し、バックエンドの LLM 呼び出しは未導入とする。[MCP 仕様](specs/review-mcp.md) と [ADR-0007](adr/0007-review-mcp.md) に責務を記載する。

Issue #50 で、`newServer` から `collectReviewsDefinition`（公開する説明・schema・annotations）と `newCollectReviewsHandler`（HTTP／SDK のキャンセル連携と終了時の解放、`collect` の呼び出し）を分離した。`newServer` は Server の生成・設定・登録を担当する。同じ MCP adapter 内の責務を整理したもので、収集ユースケースや新しい通信境界は増やさない。

Issue #52 で、JSON 読み取りを MCP 受信型の `toolInput.UnmarshalJSON`、MCP 用の既定値・上限と共通正規化による収集条件の構築を `normalizeToolInput` に分離した。許可する項目名は受信型の JSON タグから取得する。`parseInput` は標準 `json.Unmarshal` と条件構築を順に呼び、既存の結果と失敗時のゼロ値・原因を維持する。いずれも MCP の入力変換であり、通信や利用枠の操作を含まない。MCP の小さい上限を収集ユースケースや CLI へ移さない。

MCP の開始回数と同時数は handler が所有する入口の制限とし、収集操作・CLI へ混ぜない。limiter は受付可否と型付きの拒否理由を返し、failure.go の usageFailure が MCP の公開コードと説明文へ変換する。[利用制限仕様](specs/mcp-usage-limits.md) と [ADR-0008](adr/0008-mcp-usage-limits.md) に従う。

Issue #54 で、利用制限の設定は未検証の `LimitsConfig` から `NewLimits` で生成する構成にした。環境変数の解析は main 側、正数検査と設定値のコピー保持は生成 API が担当し、`NewHandler` は未初期化のゼロ値だけを拒否する。生成済みの設定と handler の可変な受付状態を区別する。設定の生成・取得は通信や利用枠の操作を行わず、新たな収集ユースケースではない。

Issue #40 で MCP の公開用失敗情報と結果生成を収集の実行手順から分離した。`failure.go` が原因の分類と案内、`result.go` が JSON・分析ガイド・応答量・原因保持を担当する。入力検査は adapter の `collect`、context・利用枠と収集の順序は mcpapplication.CollectReviews に置き、収集ユースケースや新しい通信境界は増やさない。

Issue #35 の `analysis_guide` は MCP の案内と結果変換の責務であり、新たな分析ユースケースではない。共通本文をバイナリへ埋め込み、収集結果とともに提供する。steam・CLI・保存先へ分析方針を混ぜず、分析は接続元の AI、採否は人間に残す。[レビュー分析仕様](specs/review-analysis.md) に従う。

`mcpaccess` の内部は domain と adapter に分ける。公開 URL と許可 Host のルールは mcpaccess/domain の `Access`、HTTP の保護は mcpaccess/adapter の `Protect`、ローカル専用時の保護は Issue #48 の `ProtectLocal` に分離する。main が公開設定を組み立てて `reviewmcp.NewHandler` へ渡し、`NewHandler` がモードに応じた保護を選ぶ。未指定はローカル専用、指定時は匿名公開とし、認証・JWT・OAuth metadata は提供しない。収集ユースケース・CLI へ公開設定を混ぜない。[公開 MCP 仕様](specs/mcp-public-access.md) と [ADR-0009](adr/0009-public-mcp-access.md) に従う。

## 評価種別（Issue #33）

Issue #43 では `PageOptions` と `CollectionOptions` の `ReviewType` は独立した数値列挙型 ReviewType の0 / 1 / 2で表す。値は公式の定数から定義し、HTTP adapter で EUserReviewsReviewType に変換する。文字列は CLI・MCP の入力境界で ParseReviewType により解析し、JSON 出力境界の FormatReviewType で戻す。未知の数値は共通正規化と送信・出力で拒否する。収集は全ページへ条件を渡し、Collection と JSON の options.review_type に保持する。CLI の --review-type と MCP の review_type は同じ操作へ接続し、独自のページングや取得後の間引きを追加しない。保存先の interface とバックエンド分析の未導入状態は維持する。

## Steam の数値型（Issue #43）

AppID・RecommendationID・TimestampCreated・TimestampUpdated は steam/domain の定義型を各操作と呼び出し元へ引き継ぐ。評価種別の列挙型は内部全体で引き継ぎ、残る2列挙型は HTTP adapter の数値表現に使う。領域の型に JSON の読み書きを持たせず、HTTP の受信と jsonoutput の出力で変換する。選択理由と外部形式は [ADR-0012](adr/0012-steam-domain-types.md) に従う。

## レイヤー内の配置（Issue #56）

- steam/domain は型・API 列挙値・純粋な正規化と収集状態、steam/application は操作と PageFetcher・CollectionSaver、steam/adapter は HTTP と要求・応答の変換を置く。ReviewType の基底型と定数の定義元は維持した。
- reviewmcp/domain は Limits・Limiter と受付・返却、application は収集の実行と期限、adapter は HTTP/SDK と入力・結果・ガイドを置く。入力不正は取得と受付の前に拒否し、拒否時の JSON と取得失敗の原因保持を維持する。
- reviewcli/adapter と jsonoutput/adapter は入口と出力の既存の責務を維持する。空の domain/application は追加しない。
- mcpaccess/domain の Access は IsInitialized・AllowsHost で設定と照合を提供し、adapter.Protect が設定と next を受け取って HTTP を保護する。内部の authority 表現は公開しない。
- cmd/server と cmd/collect-reviews は main の起動順序・具体的な依存の所有と終了を維持し、既存の I/O と表示をそれぞれの adapter へ置く。
