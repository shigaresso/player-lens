# レビュー収集 MCP

reviewmcp の内部は domain（上限と受付）、application（期限・枠と収集）、adapter（HTTP/SDK・入力・応答）に分ける。Issue #77 では Controller を adapter/handler/collect_reviews.go に、入力・応答・schema・ガイドの詳細を handler/protocol に置く。reviewmcp.NewHandler と newServer は外側の adapter/server.go、ツール定義は adapter/collection_definition.go に配置する。配置は [ADR-0013](../adr/0013-domain-folder-layers.md)、対応は [Controller 一覧](../handler-routing.md) を参照する。

## 目的と状態

[Issue #25](https://github.com/shigaresso/player-lens/issues/25) で `/tools` を導入する。ChatGPT が分析を担当し、Go は収集結果をメモリから返す。LLM API キー、LLM 呼び出し、レビューのファイル・DB 保存は使用しない。ChatGPT の会話保存は PlayerLens の保存処理と別である。実装・検証の状態は [作業記録](../worklogs/25-review-mcp.md) に残す。

## 接続と入力

公式 Go SDK v1.8.0 の Streamable HTTP を使う。stateless・JSON 応答とし、GET の SSE 待機や永続セッションを使わない。SDK の対応版 2026-07-28 の discover と旧版 2025-11-25・2025-06-18・2025-03-26 の initialize を検証する。`/tools` だけを接続先とし、旧 `/mcp`・`/tools/`・`/tools/child` は404で拒否する。healthz の契約は維持する。

ツールは `collect_reviews` の1つとし、読み取り専用・外部通信ありと宣言する。

| 項目 | 契約 |
| --- | --- |
| `app_id` | 必須の整数、1〜4294967295 |
| `review_type` | 任意の文字列。省略・空なら all。all・positive・negative のみ |
| `language` | 省略・空なら english。既存の言語正規化に従う |
| `page_size` | 省略・0なら100、1〜100 |
| `max_reviews` | 省略・0なら20、1〜100 |
| `max_pages` | 省略・0なら3、1〜5 |

Issue #148 で任意項目の入力 schema に実際の正規化と一致する default を掲載し、自動テストで照合する。トップページはこの schema の数値上限・既定値・評価 enum を利用する。schema の注釈追加で入力の受理・拒否や収集上限は変更しない。

評価種別は入力 schema に enum と説明を掲載し、共通の正規化で検査する。null・文字列以外・大文字・空白・未知値は利用回数を消費する前に拒否する。正常・部分失敗・利用制限の応答にも指定した評価種別を保持する。

負数・型違い・範囲外・未知の項目・非オブジェクトは `invalid_input` とし、Steam 通信を行わない。カーソル・重複・終了条件は `steamapplication.CollectReviews` に任せる。MCP のより小さい上限は入口で適用し、CLI と収集操作の既存の上限は変えない。

### 入力変換の境界（Issue #52）

`protocol.ParseInput` は JSON の読み取りと収集条件の構築を順に行い、成功時は AppID と生成済みの有効な CollectionOptions、失敗時はゼロ値と `steamdomain.ErrInvalidArgument` を保持する既存のエラーを返す。

`tool_input.go` の `toolInput.UnmarshalJSON` が JSON のオブジェクト・正確な項目名・null・型・型に収まる数値を検査する。許可する項目名は `toolInput` の明示した JSON タグから初期化時に一度だけ取得し、以後は変更しない。項目名の別の一覧や `switch` を持たない。受信型への読み取りであり、steam の領域型へ JSON 処理を追加しない。

`protocol.ParseInput` は標準の `json.Unmarshal` で読み取ってから条件を構築する。JSON 構文と末尾の不正は標準 Unmarshal が拒否し、メソッド内部も入力全体を検査する。2回目の Decode や DisallowUnknownFields の設定は置かない。メソッドを持たない同じフィールドの一時値へ読み、成功時だけ受信先へ反映する。失敗時は受信先を変更せず、省略項目は以前の値を引き継がずゼロ値にする。標準 Unmarshal が先に構文エラーを返す場合も、`protocol.ParseInput` は既存と同じゼロ値と `steamdomain.ErrInvalidArgument` を保持するエラーへ変換する。読み取り段階では既定値・MCP 上限・言語や評価種別の条件を適用しない。

Issue #133 で入力失敗の診断原因を保持する。JSON の構文・型の失敗は `errors.As`、入力不正は `errors.Is(err, steamdomain.ErrInvalidArgument)` で識別できるようにする。オブジェクトでない入力・未知の項目・null と、App ID・件数・ページ数・評価種別・言語などの拒否理由を内部のエラーへ残す。条件生成が返した原因は置き換えずに引き継ぐ。未知の項目名や入力値・本文全体を独自の診断文へ転載せず、既知の項目名と拒否理由を説明する。標準 JSON エラー自体は入力の一部を含む場合があるため、診断用エラーをそのままクライアントへ返したりログへ出力したりしない。この変更は入力の受理・拒否、大小文字の区別、既定値を変えない。

`input.go` は MCP 用の既定値・上限定数、`ParseInput`、`normalizeToolInput` の順に配置し、解析の手順と条件生成を同じファイルで読めるようにする（Issue #115）。関数の責務は分離したまま維持し、独自の JSON 読み取りは `tool_input.go` に置く。`normalizeToolInput` は `toolInput` を値で受け、App ID と MCP の件数・ページ上限を検査し、MCP 用の既定値を適用する。評価種別の文字列を `steamdomain.ParseReviewType` で変換し、`steamdomain.CollectionOptionsConfig` から `steamdomain.NewCollectionOptions` で言語・ページ件数などを検査・正規化した値を生成する。呼び出し側の入力は変更せず、失敗時はゼロ値と `steamdomain.ErrInvalidArgument` を返す。生成失敗のゼロ値を有効な既定条件として実行せず、`collectReviewsHandler.Handle` がエラーを処理して空の Collection から失敗応答を作る。生成済みの値を application と収集状態へ渡し、条件の検査を繰り返さない（[ADR-0015](../adr/0015-collection-options-boundary.md)）。両処理は通信・時刻・利用制限に依存しない。`collectReviewsHandler.Handle` による `invalid_input` 応答と、不正入力時に取得・利用枠を消費しない順序を維持する。

## 結果・失敗・資源

`structuredContent` と同じ JSON のテキストを返す。`collection` は既存の `jsonoutput.MarshalCollection` の schema_version=1 とし、レビュー ID・本文・言語・評価・UNIX 秒、条件・成功ページ数・終了理由を取得順に返す。Issue #43 の名前付き数値型は共通の JSON 変換で既存形式へ戻し、ID は0・最大値も10進文字列で返す。`returned_count` は返した件数、`omitted_reviews` は応答量超過で返せなかった件数とする。成功時の `error_code`・`message` は空である。件数・ページ上限での正常終了は全件収集を意味しない。

Issue #35 で、すべてのツール結果に `analysis_guide`（文字列の `version` と `instructions`、初版は版1）を追加した。共通本文をバイナリへ埋め込み、レビュー本文や成功・失敗で変更しない。出力 schema とツール説明に存在を案内する。ガイドも二重化した128 KiB上限の対象とし、応答量超過時にも保持する。[レビュー分析仕様](review-analysis.md) と [ADR-0011](../adr/0011-client-review-analysis-guide.md) に内容・担当・制約を定める。Collection と CLI の JSON は変えない。厳密にトップレベルの追加項目を拒否するクライアントは対応が必要である。

取得失敗は `fetch_failed`、キャンセルは `cancelled`、期限は `deadline_exceeded` とし、`isError=true` と部分結果・failed を返す。内部のエラー文字列はモデルへ返さない。レビュー本文は外部データであり、指示として採用しない旨をツール説明へ記載する。この説明だけでモデルの遵守を保証しない。

HTTP 本文は16 KiBまでとし、SDK が超過を413で拒否する。シリアライズした SDK の CallToolResult は128 KiBまで（JSON-RPC 外枠は含まない）。超過時は `response_too_large`、レビュー配列を空にした収集条件・取得ページ数・終了理由、省略件数、件数を減らす案内を返す。本文を切り詰めたレビューや完全な成功として返さず、元のレビューを外部保存しない。本文省略は一度だけ行い、省略後の応答も128 KiBを超える場合は結果を返さず、サイズ超過として識別できる変換エラーで終了する。応答を生成できない場合も収集・変換の原因をサーバー側で保持し、表示文字列に内部の収集原因を含めない。

レビュー収集用の期限は8秒。短い呼び出し側の期限・キャンセルを優先する。旧プロトコルを含め、元の HTTP リクエストと SDK の context のキャンセルを収集へ伝える。サーバーを HTTP リクエストごとに組み立て、stateless transport が一時セッションを閉じる。タイマー・context の連携 callback はハンドラーで解放する。HTTP クライアントと接続プールは main が組み立て、終了時に idle 接続を閉じる。再試行は追加しない。

Issue #84 で、MCP 用 application は開始前キャンセル／期限超過を入口で早期 return する。steam/domain.NewFailedCollection で入力条件と空配列を保持した失敗結果を作り、元の原因を MCP 用の文脈でラップして Result.Err に保持する。下位の収集をキャンセル結果の生成目的で呼ばない。受付拒否・収集中の失敗も共通の失敗結果生成を利用し、公開応答と部分結果は維持する。同時枠の保持範囲は Issue #113 で収集終了までに変更し、Result を返す前に枠を返却する。

### 入力失敗の公開情報と内部原因（Issue #133）

Controller は ParseInput のエラーを `protocol.InvalidInputResult(cause)` へ渡す。公開コードは従来の `invalid_input`、説明文は「取得条件が不正です。app_id とツールの入力上限を確認してください。」とし、入力や拒否理由によって変えない。空の Collection と分析ガイド、構造化結果と同じ JSON のテキストを維持する。

入力失敗と取得失敗は `encodeResultWithCause` で、公開応答を生成した後に SDK の `SetError` へ原因を保持する。先に本文を設定することで、SDK がエラー文字列を公開本文へ補うことを避ける。サーバー側の `GetError` から `errors.Is`・`errors.As` で原因を追跡できるが、構造化結果・テキスト・シリアライズした SDK 結果には内部原因を含めない。応答変換にも失敗した場合は、固定の表示文言を持つエラーへ変換原因と入力・取得原因の両方を保持する。

不正入力では引き続き UseCase を呼ばず、Steam 通信・利用枠を消費しない。context の連携と解放は既存の Controller の defer が担当する。Issue #133 ではログ出力・保存・OpenTelemetry を導入しなかった。Issue #135 の構造化ログでは標準 JSON エラーの全文を出さず、安全な分類のみを記録する。保存・OpenTelemetry は引き続き未導入とする。

### レビュー収集の構造化ログ（Issue #135）

`Handle` の開始から入力・UseCase・応答生成までの結果を、`review_collection.completed` として通常 return と panic の各終了時に1回記録する。msg は「MCP のレビュー収集を処理した」に固定する。SDK に返すまでの観測であり、HTTP 送信・クライアント受信の成功を意味しない。初期化・ツール一覧・未定義ツール・Host/Origin や本文上限での拒否はこのイベントに含めない。

main の JSON/stderr logger（service.name=player-lens）と計測用の時計を run → NewHandler → newServer → NewCollectReviewsHandler へ渡す。NewHandler は logger と時計が nil なら構築エラーにする。構築時には計測・ログ・収集を実行しない。計測用の時計は既存 limiter の時計と分け、利用枠の規則は変えない。通常テストは明示した破棄先、ログの契約テストはメモリ上の記録先を渡す。

Controller が実行の段階と結果をまとめ、handler/logging の ReviewCollection が属性・level・安全な分類を担当する。公開コードは protocol.CollectionErrorCode・UsageErrorCode が既存の応答分類を参照して共有する。応答の JSON やエラー文章を解析して属性を作らない。

| 属性 | 内容・省略条件 |
| --- | --- |
| event | review_collection.completed |
| service.name | 起動側の player-lens |
| duration_ms | 開始から応答生成までの数値のミリ秒。ログ出力・context 後始末・送信時間を含まない |
| outcome | success、rejected、cancelled、failed |
| failure_stage | 失敗時だけ input・collection・response_encode。panic は発生段階 |
| error_code | 失敗時だけ既存の公開分類。応答生成不能は response_encoding_failed、panic は panic |
| cause_kind | 失敗時だけ安全な固定分類 |
| app_id | 入力解析と条件生成が成功した場合だけ |
| collected_count | UseCase が受付拒否以外の結果を返した場合の、本文省略前のレビュー件数 |
| pages_fetched | 同じ条件で、空ページを含む成功ページ数 |
| stop_reason | 同じ条件で、収集結果の終了理由 |

失敗は panic → 応答生成不能 → 本文省略 → 入力不正 → 受付拒否 → 収集失敗の順に最終結果を分類する。成功・入力不正・受付拒否・キャンセルは INFO、期限超過・取得失敗・本文省略は WARN、応答生成不能・panic は ERROR とする。入力不正と受付拒否は outcome=rejected、キャンセルは cancelled、その他の失敗は failed とする。期限は収集失敗の分類であり、正常な全件取得とは扱わない。

cause_kind は errors.Is・errors.As により cancelled、deadline_exceeded、json_syntax、json_type、response_too_large、cursor_cycle、http_status、timeout、upstream_response_too_large、invalid_response、invalid_argument をこの順に判定し、残りは unknown とする。受付拒否は usage_limit、panic は panic とする。判定中に任意の Error 文字列や型名・入力値を属性へ渡さない。入力不正の条件別の文章は内部原因として保持したままとし、ログ上は安全な invalid_argument にまとめる。

入力 JSON・未知の項目名・レビュー本文・カーソル・URL・任意の内部エラー全文は記録しない。標準 JSON エラーの Value・Field なども転載しない。失敗していない属性や仮の trace ID は付けない。属性はアプリケーション固有であり、OTel の標準属性と称しない。

計測と結果記録は所有 context の cancel より前に実行し、処理中の context を LogAttrs へ渡す。呼び出し元の値とキャンセルを維持し、自分自身の cleanup を失敗原因へ分類しない。収集の同時枠は従来どおり UseCase の終了時に返却済みとなる。ログ writer の通常の Write エラーでは公開結果・原因・枠返却・context の解放を変更せず、ログの再試行は行わない。panic の値をログへ含めず、回復せず元のまま伝播させる。

新たなファイル・保存先・goroutine・buffer は本番に作らず、出力先の Close も引き受けない。stderr の保存・検索・保持期間、任意の writer の非ブロッキング、OTel の SDK・転送・trace・metrics は別途決定する。実装の検証は単体、4版の実 SDK、実バイナリの stderr と race で確認する。設計は [ADR-0021](../adr/0021-mcp-collection-structured-logging.md) を参照する。

### ツール登録と実行の境界（Issue #50）

`newServer` は Server の生成・対応プロトコルの設定・ツール登録を担当する。`collection_definition.go` の `collectReviewsDefinition` がツール名・説明・入出力 schema・annotations を構築する。各生成の可変データは独立させ、ほかの Server の定義に変更を共有しない。定義の生成では通信・収集を行わない。

`handler/collect_reviews.go` の `handler.NewCollectReviewsHandler` は、HTTP リクエストの context・取得依存・共有 limiter を受け取り、SDK から呼ばれる実行 handler を作る。Issue #80 で、非公開の collectReviewsHandler の Handle メソッド値を返す構成にした。Handle が呼び出し時に HTTP の context を親とする context を作り、SDK のキャンセルも連携させて入力・UseCase・応答変換を順に行う。終了時に連携 callback を停止して所有 context をキャンセルする。呼び出し側の context はキャンセルしない。handler 型のフィールドは HTTP ごとの固定の依存とし、実行中に書き換えない。呼び出しごとの context は Handle のローカル変数として保持し、共有可変状態に格納しない。HTTP の ServeHTTP と MCP の Handle は入口の契約が異なり、HTTP の受付とプロトコル処理は引き続き SDK に任せる。Server と定義は HTTP ごとに作り、limiter は `NewHandler` で一度だけ作って共有する寿命を維持する。

### 結果変換の境界（Issue #40）

`collectReviewsHandler.Handle` は入力解析→application.CollectReviews→protocol.EncodeResult を順に呼ぶ。Issue #113 で同期 callback を廃止し、application は利用枠・取得期限・収集を管理して既存の Result を直接返す。Collection に収集結果、Rejection に受付の拒否理由、Err に収集の原因を保持する。所有する枠と取得用 context は戻る前に解放する。Controller は結果を受け取って公開用の失敗分類と SDK 応答変換を行い、変換エラーを SDK へ返す。`handler/protocol/failure.go` の `toolFailureInfo` は公開するコードと説明文を持つ情報型で、元のエラーは別途保持する。`collectionFailure` は収集エラーをこの公開情報へ変換し、`usageFailure` は limiter の拒否理由を公開コードと説明文へ変換する。複合原因ではキャンセル・期限・取得失敗の順を維持する。

`handler/protocol/result.go` の `encodeResult` と `marshalToolResult` は公開情報を toolFailureInfo のまま受け取り、コードと説明文を別引数へ分解しない。`marshalToolResult` は Collection・公開情報・ガイドを同じ JSON の構造化結果・テキストへ変換する。`encodeResult` は省略件数を引数に受け取らず、SDK の通常応答と省略応答のサイズを判定する。`encodeOversizedResult` は省略件数を数えて本文を除いた応答を作り、`encodeResult` を再び呼ばない。`encodeResultWithCause` が入力・取得の原因をサーバー側へ保持し、応答量超過の原因も失わせない。変換は通信・時計・保存に依存しない。同時枠は収集終了時に返却済みとなり、変換中も別の収集を受け付ける。変換失敗・panic で他の収集の枠を返さない。application は SDK 型を扱わない。Issue #62 の旧保持要件は Issue #113 の [ADR-0020](../adr/0020-mcp-collection-slot-lifetime.md) で変更し、[ADR-0016](../adr/0016-mcp-response-slot-lifetime.md) は履歴として残す。

公開 URL が未指定なら HTTP Host は localhost またはループバック IP に限定する。Issue #29 で、明示した公開 URL の Host からも認証なしで利用する設定を追加した。Issue #82 で NewLocal／NewPublic が生成する Access は Host 判定だけとし、NewHTTPConfig が許可条件・拒否文言・SDK 設定をまとめる構成にした。両モードで単一の mcpaccess/adapter.Protect を使う。NewHandler は生成済み HTTPConfig を受け取り、設定済みの値をそのまま使う。未生成の HTTPConfig のゼロ値は構築エラーとする。許可されない Host は403で拒否する。[公開 MCP 仕様](mcp-public-access.md) に従う。Docker 転送では SDK の接続先アドレスによる自動 localhost 判定が働かない場合があるため、Host を明示的に検査する。cross-origin 保護と併用する。

HTTP の読み書き10秒・終了待機10秒は維持する。遅い本文の送信時間も HTTP の10秒に含まれるため、常に8秒の実行時間を使い切れるとは保証しない。JSON 応答であり長時間の SSE 接続を要求しない。

## 呼び出し回数と同時数

Issue #27 で1分6回・24時間100回・同時2件の制限を導入した。1つの handler の状態を HTTP ごとの Server と複数セッション・4版で共有する。拒否は `rate_limited`・`daily_limit_reached`・`busy` と空の結果を返し、Steam を呼ばない。開始回数の消費、枠の返却、固定窓と設定・再起動・複数台の制約は [利用制限仕様](mcp-usage-limits.md) と [ADR-0008](../adr/0008-mcp-usage-limits.md) に従う。

Issue #54 で `NewHandler` の設定は `NewLimits` または `DefaultLimits` で生成した値を受け取る構成にした。各上限の検証は生成側へ集約し、`NewHandler` は未初期化のゼロ値を拒否する。handler ごとの可変状態の所有と共有範囲は変更しない。

## デバッグと検証

ローカルの手順は [MCP デバッグ](../mcp-debug.md) を参照する。ChatGPT 設定は自動変更しない。Render 配置、クラウドからのトンネル、バックエンドの AI 実行用パスは未導入とする。公開 MCP は OAuth を使用せず、実 ChatGPT の公開接続は未確認とする。

入力・変換・失敗・期限をユニットテスト、実 SDK クライアントの接続・一覧・呼び出し・エラー・HTTP 切断を integration タグで確認する。Steam は差し替え、実 Steam・実 LLM を自動テストから呼ばない。本番イメージでは初期化・一覧・不正入力を通信なしで確認する。ChatGPT の実接続・分析回答は手動確認とし、未実施なら理由を記録する。

Issue #77 で、collect の入力不正応答は protocol.InvalidInputResult、UseCase の結果の公開表現は protocol.EncodeResult に委ねる構成にした。Issue #113 で同期 callback を廃止し、Controller は UseCase の戻り値を変換する。同時枠の保持は収集終了までとする。受信型・JSON 出力型は非公開、応答量超過は protocol.ErrResponseTooLarge で識別できる。ガイドと公開 schema の内容は変更しない。
