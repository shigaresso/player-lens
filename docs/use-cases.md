# アプリケーションのユースケース

既存の領域フォルダを維持し、その内部を domain・application・adapter に分類した。[ADR-0013](adr/0013-domain-folder-layers.md) に配置と依存を記録する。以下の steamdomain・steamapplication・steamadapter は internal/steam 配下の各レイヤー、mcpdomain・mcpapplication は internal/reviewmcp 配下の各レイヤーを指す。reviewmcp は reviewmcp/adapter、reviewcli・jsonoutput は各領域の adapter を指す。

## 導入済みの操作

| 操作 | 目的と責務 | 入力と依存 | 結果と失敗 | 呼び出し元の状態 |
| --- | --- | --- | --- | --- |
| `steamapplication.CollectReviews` | レビューを上限付きで収集し、カーソル・重複・終了条件を管理する | context、PageFetcher、App ID、CollectionOptions | Collection と終了理由。不正入力はゼロ値、途中失敗は部分結果と原因を保持したエラー | 収集・保存操作、MCP、テストと手動確認から呼び出した |
| `steamapplication.CollectAndSaveReviews` | 収集と結果保存の順序、部分結果・複合失敗・保存期限を管理する | context、PageFetcher、CollectionSaver、App ID、CollectionOptions | Collection と収集/保存の原因を保持したエラー。保存は一度だけ行う | CLI から接続した。MCP は保存しないためこの操作を使わない。Web は未接続 |
| `mcpapplication.CollectReviews` | MCP の開始回数・同時枠と8秒の期限を管理し、共通の収集操作を実行する | context、PageFetcher、解析済みの App ID と CollectionOptions、共有 Limiter | Result に収集結果・拒否理由・元の失敗を保持して直接返す。戻る前に枠と取得用 context を解放し、応答変換は Controller が続けて行う | MCP adapter が入力検査後に呼び出す |

[収集仕様](specs/collect-reviews.md) と [ADR-0005](adr/0005-bounded-review-collection.md) に契約と判断を記載する。

## パーツとの境界

`steamadapter.Client.FetchReviewsPage` は1回の HTTP 通信と入力・応答の検査、応答 Body の解放を担当する adapter とする。収集操作とは区別し、ページングや収集全体の件数・時間・部分失敗を重複させない。

Issue #88 で、HTTP リクエストの組み立てを同じ adapter の非公開関数 `newReviewPageHTTPRequest` にまとめた。API 入力・JSON・クエリ・context 付き GET を構築し、通信や期限の生成は行わない。FetchReviewsPage は入力検査、取得用 context の生成と解放、要求構築の呼び出し、送信と応答処理の呼び出しを担当する。既存の変換・入力・通信・資源解放の契約を維持し、新しいユースケースは増やさない。

Issue #38 で adapter 内のリクエスト変換を分離した。Issue #43 では文字列の評価種別から数値への変換を入力境界へ移した。`newReviewPageRequest`、`encodeReviewPageQuery` は、正規化済みの条件を Steam API の入力とクエリへ変換する通信なしの処理であり、新しいユースケースではない。 Issue #64 で評価種別の検査を FetchReviewsPage が呼ぶ NormalizePageOptions に集約し、newReviewPageRequest は正の App ID と検査後の条件の変換だけを担当する構成にした。private の重複検査と error 戻り値を除き、公開入口で不正入力を通信前に拒否する契約を維持した。

Issue #39 で応答解釈も分離した。`decodeReviewPage` は本文と取得条件からページを作り、Issue #43 の `parseRecommendationID` は ID の文字列を0を含む uint64 に変換する。Issue #66 で、Body の読み取り・サイズ制限・解釈の呼び出し・Close を同じ adapter の `readReviewPageResponse` にまとめた。FetchReviewsPage は送信と context を担当し、応答処理が Close を終えた後で context を解放する。JSON の解釈は decodeReviewPage に維持し、新しいユースケースやレイヤーは増やさない。

Issue #68 で、レビュー1件の ID と項目の変換を convertReview にまとめた。decodeReviewPage はページ全体の検査・変換の呼び出し・組み立てを担当し、変換に失敗したページは部分的に返さない。両処理は同じ adapter 内の純粋な処理とする。

Issue #58 で、重複・カーソル・成功ページ数・終了判断を steam/domain の CollectionState にまとめた。状態生成時に App ID を検査し、生成済みの CollectionOptions から設定のコピーを保持する。条件の検査は Issue #60 で NewCollectionOptions へ移し、状態では繰り返さない。NextPage と AddPage で続行と状態更新を扱う。CollectReviews は取得・期限・キャンセルと失敗の原因、Fail による失敗確定を担当する。Result はレビュー配列もコピーして返す。状態は通信や context を持たず、新しいユースケースではない。[ADR-0014](adr/0014-steam-collection-state.md) に境界を記録する。

`NormalizePageOptions` は1ページの取得条件を検査・正規化する。収集条件は `NewCollectionOptions` が未検証の `CollectionOptionsConfig` から生成し、検査・正規化とコピーの保持を担当する。`CollectionOptions` のゼロ値も有効な既定条件で、`Config()` はコピーを返す。結果の `Collection.Options` は表示・保存用の Config とし、空の Collection の未指定項目を保つ。旧公開 NormalizeCollectionOptions は残さず、条件が不正なテストの契約を生成 API・CLI/MCP の入口へ引き継ぐ。[ADR-0015](adr/0015-collection-options-boundary.md) に境界を記録する。`jsonoutput.MarshalCollection` は通信しない JSON 変換、`jsonoutput.Saver` は writer への出力を担当する。DB の永続保存・分析・レポート生成は未導入とする。

入口側が HTTP クライアントと `Client` を組み立て、同じ `CollectReviews` へ渡す。操作内に具体的な HTTP クライアントや DB を固定せず、同じ Go アプリケーション内の呼び出しに HTTP・MCP 通信を挟まない。

Issue #92 で、jsonoutput/adapter の非公開関数 newCollectionDocument に出力用データの構築をまとめ、MarshalCollection はその構築と JSON 化を順に呼ぶ構成にした。取得条件とレビューは項目名で対応を明示する。両処理は通信・保存を行わず、既存の出力と失敗・入力の非変更を維持する。MCP と Saver の呼び出し元や公開 API、新しいユースケースは増やさない。

CLI の `reviewcli.Run` は引数の解析と `CollectAndSaveReviews` への接続を担当し、収集や保存の手順を重複させない。main が JSON Saver を渡す。保存先を変更するときは同じ CollectionSaver 契約を実装して組み立て箇所で選び、操作と入力解析を共通で使う。[CLI 仕様](specs/review-cli.md) と [ADR-0006](adr/0006-collection-output-boundary.md) に責務を記載する。

MCP の `reviewmcp.NewHandler` は HTTP/SDK と共有する制限状態を組み立てる。adapter は入力・HTTP 制限と Collection の応答変換、mcpapplication.CollectReviews は利用枠と実行期限を担当する。application から steamapplication.CollectReviews を呼び、adapter は JSON の純粋変換だけを再利用する。保存操作は呼ばない。分析は接続元の ChatGPT が担当し、バックエンドの LLM 呼び出しは未導入とする。[MCP 仕様](specs/review-mcp.md) と [ADR-0007](adr/0007-review-mcp.md) に責務を記載する。

Issue #50 で、`newServer` から `collectReviewsDefinition`（公開する説明・schema・annotations）と `newCollectReviewsHandler`（当時の HTTP／SDK のキャンセル連携と終了時の解放、`collect` の呼び出し。Issue #80 で名前付き Handle へ統合）を分離した。`newServer` は Server の生成・設定・登録を担当する。同じ MCP adapter 内の責務を整理したもので、収集ユースケースや新しい通信境界は増やさない。

Issue #52 で、JSON 読み取りを MCP 受信型の `toolInput.UnmarshalJSON`、MCP 用の既定値・上限と NewCollectionOptions による収集条件の生成を `normalizeToolInput` に分離した。許可する項目名は受信型の JSON タグから取得する。`parseInput` は標準 `json.Unmarshal` と条件構築を順に呼び、既存の結果と失敗時のゼロ値・原因を維持する。いずれも MCP の入力変換であり、通信や利用枠の操作を含まない。MCP の小さい上限を収集ユースケースや CLI へ移さない。

MCP の開始回数と同時数は handler が所有する入口の制限とし、収集操作・CLI へ混ぜない。limiter は受付可否と型付きの拒否理由を返し、failure.go の usageFailure が MCP の公開コードと説明文へ変換する。[利用制限仕様](specs/mcp-usage-limits.md) と [ADR-0008](adr/0008-mcp-usage-limits.md) に従う。

Issue #54 で、利用制限の設定は未検証の `LimitsConfig` から `NewLimits` で生成する構成にした。環境変数の解析は main 側、正数検査と設定値のコピー保持は生成 API が担当し、`NewHandler` は未初期化のゼロ値だけを拒否する。生成済みの設定と handler の可変な受付状態を区別する。設定の生成・取得は通信や利用枠の操作を行わず、新たな収集ユースケースではない。

Issue #62 では [ADR-0016](adr/0016-mcp-response-slot-lifetime.md) に従い、同期 callback の応答変換完了後に同時枠を返す構成へ修復した。Issue #113 の [ADR-0020](adr/0020-mcp-collection-slot-lifetime.md) で目的を収集の同時実行制限に絞り、callback を廃止する。application は収集終了時に枠と取得用 context を解放し、既存の Result を直接返す。Controller が結果を受けて JSON・SDK 応答を生成し、application は SDK 型を含めない。開始前キャンセル・受付拒否では未確保の枠を返さず、収集の原因は Result.Err に保持する。

Issue #86 で、CollectionState.AddPage の重複排除を非公開メソッド appendUnseenReviews に分離した。既読 ID とレビュー配列の更新を同じ場所に保ち、AddPage からはページ反映と終了判断の手順を読む。操作・通信・公開 API と終了優先順は変更していない。

Issue #84 で、MCP 用 CollectReviews の開始前キャンセル／期限超過は、共通の NewFailedCollection で空の失敗結果を作って早期 return する構成にした。Issue #113 では callback を削除し、同じ失敗 Result を直接返す。下位の収集・期限の生成・利用枠の確保へ進まない。CollectionState の取得・context・循環失敗と受付拒否も共通生成を使い、入力条件・部分結果を維持する。枠の確保と返却は application に残し、Issue #113 からは保持範囲を収集終了までにする。

Issue #40 で MCP の公開用失敗情報と結果生成を収集の実行手順から分離した。`failure.go` が原因の分類と案内、`result.go` が JSON・分析ガイド・応答量・原因保持を担当する。入力検査は adapter の `collectReviewsHandler.Handle`、context・利用枠と収集の順序は mcpapplication.CollectReviews に置き、収集ユースケースや新しい通信境界は増やさない。

Issue #35 の `analysis_guide` は MCP の案内と結果変換の責務であり、新たな分析ユースケースではない。共通本文をバイナリへ埋め込み、収集結果とともに提供する。steam・CLI・保存先へ分析方針を混ぜず、分析は接続元の AI、採否は人間に残す。[レビュー分析仕様](specs/review-analysis.md) に従う。

`mcpaccess` の内部は domain と adapter に分ける。公開 URL の検査と両モードの Host 判定は mcpaccess/domain の `Access`、HTTP の保護は mcpaccess/adapter の `Protect` に分離する。Issue #82 で `NewLocal`／`NewPublic` から生成する Access は Host 判定だけとした。reviewmcp/adapter の `NewHTTPConfig` が許可条件・拒否文言・SDK 設定を組み立て、main は生成済みの HTTPConfig を `reviewmcp.NewHandler` に渡す。NewHandler は常に条件と文言を Protect に渡し、利用側でモードを選び直さない。未指定はローカル専用、指定時は匿名公開とし、認証・JWT・OAuth metadata は提供しない。収集ユースケース・CLI へ公開設定を混ぜない。[公開 MCP 仕様](specs/mcp-public-access.md) と [ADR-0009](adr/0009-public-mcp-access.md) に従う。

## 評価種別（Issue #33）

Issue #43 では `PageOptions` と `CollectionOptionsConfig` の `ReviewType` は独立した数値列挙型 ReviewType の0 / 1 / 2で表す。値は公式の定数から定義し、HTTP adapter で EUserReviewsReviewType に変換する。文字列は CLI・MCP の入力境界で ParseReviewType により解析し、JSON 出力境界の FormatReviewType で戻す。未知の数値は共通正規化と送信・出力で拒否する。収集は全ページへ条件を渡し、Collection と JSON の options.review_type に保持する。CLI の --review-type と MCP の review_type は同じ操作へ接続し、独自のページングや取得後の間引きを追加しない。保存先の interface とバックエンド分析の未導入状態は維持する。

## Steam の数値型（Issue #43）

AppID・RecommendationID・TimestampCreated・TimestampUpdated は steam/domain の定義型を各操作と呼び出し元へ引き継ぐ。評価種別の列挙型は内部全体で引き継ぎ、残る2列挙型は HTTP adapter の数値表現に使う。領域の型に JSON の読み書きを持たせず、HTTP の受信と jsonoutput の出力で変換する。選択理由と外部形式は [ADR-0012](adr/0012-steam-domain-types.md) に従う。

## レイヤー内の配置（Issue #56）

- steam/domain は型・API 列挙値・純粋な正規化と収集状態、steam/application は操作と PageFetcher・CollectionSaver、steam/adapter は HTTP と要求・応答の変換を置く。ReviewType の基底型と定数の定義元は維持した。
- reviewmcp/domain は Limits・Limiter と受付・返却、application は収集の実行と期限、adapter は HTTP/SDK と入力・結果・ガイドを置く。入力不正は取得と受付の前に拒否し、拒否時の JSON と取得失敗の原因保持を維持する。
- reviewcli/adapter と jsonoutput/adapter は入口と出力の既存の責務を維持する。空の domain/application は追加しない。
- mcpaccess/domain の Access は非公開のローカル用・公開用の値型が満たす AllowsHost だけの共通契約とし、adapter の Protect が設定と next を受け取って HTTP を保護する。内部の authority 表現は公開しない。
- cmd/server と cmd/collect-reviews は main の起動順序・具体的な依存の所有と終了を維持し、既存の I/O と表示をそれぞれの adapter へ置く。

## Controller の配置（Issue #77）

handler 直下の本番ファイルは、リクエストの準備・入力・UseCase・応答という操作全体の流れを読む Controller の一覧とする。MCP は Issue #80 で handler/collect_reviews.go の collectReviewsHandler.Handle に context の連携・入力・UseCase・応答・後始末をまとめた。NewCollectReviewsHandler は HTTP ごとの固定の依存を持つ handler のメソッド値を SDK へ渡し、実行ごとの context を共有状態へ格納しない。入力・応答・schema・ガイドの詳細は handler/protocol に置く。SDK の構築・共有状態・登録は外側の adapter に、Host/Origin の middleware は mcpaccess/adapter に維持する。

healthz は handler/health.go と下位の logging に分ける。Issue #80 で healthHandler.Handle と collectReviewsHandler.Handle に実行入口名を揃えた。healthz の生成関数は標準 http.HandlerFunc、MCP の生成関数は SDK に適合するメソッド値を返す。操作と期限の責務を維持する。同時枠の保持範囲は Issue #113 で収集終了までに変更し、Controller が続けて応答変換を行う。新しいユースケースは追加せず、[Controller とルーティングの一覧](handler-routing.md) から登録と各処理を追えるようにする。
