# 最小 HTTP サーバーの仕様

## 目的と状態

[Issue #11](https://github.com/shigaresso/player-lens/issues/11) で Go・Compose と HTTP の起動基盤を導入する。実装・ローカル検証の結果は Issue と作業記録へ記載し、Render に配置済みとは扱わない。

## HTTP の契約

- `GET /healthz` は200、`Content-Type: text/plain; charset=utf-8`、本文 `ok` と改行を返す。`HEAD /healthz` は同じヘッダーと200を返し、本文を返さない。
- 未定義のパスは404、`/healthz` の GET・HEAD 以外のメソッドは405とする。`/healthz/child` などはヘルスチェックに含めない。
- 起動した HTTP サーバーの応答を確認するものであり、DB・Steam・LLM・MCP の接続や調査の正常性を示さない。収集操作の一覧は [ユースケース](../use-cases.md)、追加した `/tools` の契約は [MCP 仕様](review-mcp.md) を参照する。
- 応答の書き込みに失敗した場合は操作と安全な原因分類を構造化ログへ記録する。接続へ書き込めない状況で新たな正常応答を作らない。

## 起動時の生成と経路登録（Issue #74）

main は logger とプロセスの context を生成し、run は各 handler とルーターを組み立てる。server adapter の NewHealthHandler(logger, now) は生存確認の handler を生成する。生成時にはリクエスト処理・時刻取得・ログ出力を行わない。MCP の handler は従来どおり reviewmcp.NewHandler で生成する。

run は http.NewServeMux でルーターを生成し、RegisterRoutes(mux, healthHandler, mcpHandler) で経路を登録してから待ち受けを開始する。RegisterRoutes は既存の mux に生成済み handler を登録するだけで、依存や handler・ルーターを生成せず、handler を実行しない。無関係な既存経路は維持する。旧 server adapter の NewHandler は残さない。

RegisterRoutes は GET /healthz（HEAD を含む）と、MCP が非 nil の場合の /tools を登録する。mux と healthHandler は有効な非 nil の値を渡し、登録は受付開始前に行う。競合するパターンは事前に登録しない。重複登録などは標準 ServeMux の panic に従い、独自の検査や復旧は追加しない。各 handler は並行利用可能なものを渡す。

Issue #76 で、登録パターンのパスとログの http.route は同じ非公開の healthPath 定義を参照する。公開経路は /healthz のまま維持し、受信 URL・query を経路の定義へ取り込まない。公開契約を検証するテストの期待値はこの実装定数を参照せず、/healthz のリテラルで確認する。

生成は New、既存の対象への登録は Register という名前で区別する。業務操作を持たない生存確認のための application/domain や、生成と登録をまとめ直す互換ラッパーは作らない。

```text
起動時:
  main: JSON logger と context を生成 → run へ渡す
  run: MCP handler と health handler を生成
       → ルーターを生成 → RegisterRoutes で登録
       → listener を生成 → Serve で受付と終了を管理

リクエスト時:
  ルーター → healthHandler.ServeHTTP
           → 開始時刻 → 応答ヘッダー → GET の本文書き込み
           → 所要時間と書き込み結果 → healthCheckLog → slog
```

起動時の組み立てはリクエストごとに実行しない。リクエストのメソッドとパスの制限はルーター、応答と計測は healthHandler、ログ項目とレベル・原因分類は healthCheckLog が担当する。healthz の応答処理完了とクライアントへの配信完了を区別する。

## 生存確認の構造化ログ（Issue #72）

起動側は slog の JSONHandler で stderr に出力し、service.name=player-lens を付与する。生成した slog.Logger と時刻取得関数を serveradapter.NewHealthHandler に渡し、handler はその依存を共有する。依存は有効な非 nil のものを渡し、同時実行時には並行利用に対応する。グローバル logger の差し替えは行わない。

GET/HEAD /healthz が通常 return したときに health_check.completed を1回記録する。時刻・level・msg は slog の標準項目とし、msg は「ヘルスチェックを処理した」に固定する。

| 項目 | 契約 |
| --- | --- |
| event | health_check.completed |
| http.request.method | GET または HEAD |
| http.route | /healthz の固定値。query は含めない |
| http.response.status_code | handler が選択する200。クライアントの受信成功を示すものではない |
| duration_ms | handler 開始から応答処理の終了までの数値（ミリ秒）。ログの出力時間とクライアントへの配信完了は含まない |
| outcome | 成功は success、本文の書き込み失敗は write_failed |
| error_code | 書き込み失敗時だけ response_write_failed |
| cause_kind | 書き込み失敗時だけ、安全な原因分類 |

成功は INFO、書き込み失敗は ERROR とする。cause_kind は errors.Is による cancelled・deadline_exceeded・connection_closed、net.Error の Timeout による timeout、残りを unknown とする。この順で判定する。切断は io.ErrClosedPipe・net.ErrClosed・syscall.EPIPE・syscall.ECONNRESET を含む。以前の任意のエラー文字列の出力を分類へ変更し、未加工の原因・リクエストの query・ヘッダー・本文を記録しない。発生していないエラー属性や仮の trace ID は付けない。

書き込み失敗では新しいステータスや本文を書き直さず、既存の応答処理を維持する。404/405・未定義パス・MCP への委譲はこのイベントに含めない。panic は従来どおり伝播し、独自の回復や panic 専用の完了ログは追加しない。生存確認の成功は Steam・MCP・調査の成功を示さない。

LogAttrs に request の context をそのまま渡す。キャンセル済みでも別の Background に置き換えない。将来の OTel bridge が参照できる経路を保持するものであり、今回は SDK・トレース・メトリクスを導入しない。

ログ出力先の通常の Write エラーで HTTP 応答を変更しない。任意の writer の非ブロッキング・保存成功・保持期間は保証しない。新しい buffer・goroutine・ファイル・DB は作らず、出力先の Close は引き受けない。stderr の保持と検索基盤は配置側の別の契約とする。

起動側が従来出していた起動と実行失敗も、同じ logger の server.started（INFO、address）と server.failed（ERROR、error）で JSON 化する。起動失敗の既存の運用診断は error に保持する。これらは別イベントであり、生存確認の属性を付けない。SDK・net/http 自身の診断を全て JSON 化する契約は含まない。

配置と比較は [ADR-0017](../adr/0017-health-structured-logging.md) を参照する。

## 設定と終了の契約

- `PORT` が空なら8080とする。ASCII の数字のみで表された1〜65535を受け付け、先頭のゼロは正規化する。全インターフェースで待ち受ける。
- 不正な設定・待ち受け失敗・予期しないサーバーエラーは操作の文脈を含むエラーと非0終了で報告する。
- 読み取りヘッダーは5秒、全読み取り・書き込みは10秒、アイドルは60秒を上限とする。
- SIGINT/SIGTERM で新規受付を停止する。処理中のリクエストは最大10秒待ち、期限内に終了した場合は正常終了する。期限超過では接続を閉じて非0終了する。
- 起動処理が listener を作成し、サーバー処理へ所有権を渡す。サーバー処理は終了時に listener・接続を閉じ、Serve の goroutine の終了を待つ。強制終了時の handler はリクエスト context のキャンセルに従う。

終了待機の上限は Serve の shutdownTimeout 引数で渡し、run は同名の定数で10秒を指定する。通常リクエストの読み取り・書き込み期限とは独立した設定であり、今回の改名と定数化では値や終了手順を変更しない（Issue #76）。

### Steam クライアントの構築と所有

サーバー起動の `run` は、server adapter の公開関数 `NewSteamFetcher` から取得クライアントと idle 接続の終了関数を受け取る。構築関数が既定の Transport を複製し、HTTP タイムアウト10秒と公式の Steam 接続先を使って組み立てる。構築中は通信しない。

構築失敗時は構築関数が Transport の idle 接続を解放し、nil のクライアント・nil の終了関数と元のエラーを返す。成功時は `run` が直後に終了関数を defer へ登録し、後続の起動失敗またはサーバー実行の終了時に呼ぶ。終了関数は idle 接続だけを閉じ、処理中のリクエストのキャンセルや HTTP サーバーの終了待機は既存の担当に残す。

## 開発時の公開ポート

- 開発用の起動入口は `python3 scripts/dev.py` とする。Docker Compose が解決した設定から `DEV_PORT` を開始値として取得する。シェル環境変数・`.env` の優先順位は Compose に従い、未指定なら8080とする。
- 公開先は `127.0.0.1` の TCP ポート1つとし、開始値は1〜65535とする。使用中なら1ずつ増やし、65535まで順に試す。選択値は子プロセスの環境変数として渡し、`.env` は書き換えない。コンテナ内と本番の `PORT` は自動変更しない。
- Docker のホストポート割り当て失敗だけを再試行する。ビルド・設定・Docker 接続・healthcheck の失敗は非0終了とし、上限まで使用中なら空きポートなしとして非0終了する。イメージのビルドは起動時に1回行う。
- 同じ Compose プロジェクトの `app` が実行中なら、二重起動を拒否し既存サービスを維持する。同じプロジェクトの同時起動は扱わず、ローカル Docker を対象とする。
- 起動と healthcheck が成功した URL を表示し、Compose Watch でソース同期・再起動を提供する。ログは `docker compose logs --follow app` で確認する。起動入口は自分が起動した `app` の停止を担当し、Ctrl+C と失敗時に停止する。コンテナ・ネットワークの削除は `docker compose down`、キャッシュ volume は保持する。

## 検証との対応

生成と登録の境界はユニットテストで確認する。health handler の生成時に時計・ログを実行しないことと、登録時に handler を呼ばないこと、渡した health/MCP handler への委譲、既存経路の保持を確認する。旧ルーター経由の応答テストは生成済み handler と明示的な登録へ移行し、ログの契約は生成済み health handler でも確認する。

routes_test.go の TestRegisterRoutes は渡した handler への登録と委譲、TestHealthRoutes は実 health handler とルーターを接続した公開応答を確認する。health_test.go の TestHealthWriteFailure は本文書き込み失敗の応答とログを確認する。Issue #76 で接続テストの名前と配置をその責務に揃えた。

設定と HTTP 応答はネットワークを使わないユニットテストで確認する。構造化ログは制御した時計とメモリ上の出力先で正常・書き込み失敗・ログ出力失敗・context の引き継ぎと対象経路を確認する。実 HTTP の GET/HEAD と JSON の接続、実バイナリの起動失敗と stderr・終了コードは integration タグで確認する。実際の待ち受け・応答・処理中の終了・期限超過・bind 失敗は `integration` タグの結合テストで確認し、race detector を実行する。本番用イメージの PORT・不正設定・HTTP 応答・SIGTERM は独立したスモーク検査で確認する。

`NewSteamFetcher` の終了関数は、実 Steam へ接続しないローカル TLS サーバーとの結合テストで確認する。取得に同じ接続を再利用できることと、返された終了関数がその接続を閉じることを観測する。テストで差し替える既定 Transport は終了時に復元し、並列実行による干渉を避ける。

開発用のポート選択・失敗伝播・二重起動拒否・停止は Docker やネットワークを使わない Python テストで確認する。連続ポートの使用中からの起動、Watch と停止はローカル Docker で実行して確認する。

Compose と本番用イメージの相違、固定版の更新、配置の未導入事項は [ADR-0001](../adr/0001-runtime-and-deployment.md) と [開発手順](../development.md) に従う。
