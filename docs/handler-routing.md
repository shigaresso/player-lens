# 受付とルーティングの一覧

## 目的と配置規則

Issue #77 で、ルーティングから割り当てられる自前の処理と受付保護を各領域の adapter/handler に揃える。domain・application・adapter は維持し、handler を受付処理の固定した配置先とする。判断は [ADR-0013](adr/0013-domain-folder-layers.md) に記録する。実装と検証の状態は Issue と作業記録へ分ける。

| 登録・接続元 | 割り当て先・処理 | 自前の入口と配置先 |
| --- | --- | --- |
| cmd/server/adapter/routes.go の GET /healthz（HEAD を含む） | 生存応答、計測、構造化ログ | cmd/server/adapter/handler/health.go の healthHandler.ServeHTTP。生成は NewHealthHandler |
| 同じ RegisterRoutes の /tools | MCP の HTTP 受付 | internal/reviewmcp/adapter/handler/handler.go の NewHandler が SDK の StreamableHTTPHandler を組み立てる |
| MCP の公開設定の有無による next の接続 | Host と cross-origin の受付保護 | internal/mcpaccess/adapter/handler の Protect／ProtectLocal が SDK handler の手前を保護する |
| SDK の HTTP リクエストごとの Server 構築 callback | newServer が設定とツール登録を行う | internal/reviewmcp/adapter/handler/handler.go の newServer。構築と登録を担当する補助関数 |
| newServer の AddTool と SDK の collect_reviews へのディスパッチ | 入力検査、ユースケースへの接続、結果変換 | internal/reviewmcp/adapter/handler/collection_handler.go の newCollectReviewsHandler から collection.go の collect へ接続する |

## 登録・処理・補助処理の境界

HTTP の経路一覧は cmd/server/adapter/routes.go に置く。run が health/MCP handler とルーターを生成し、RegisterRoutes が登録する。登録時に handler を実行しない。healthz の登録パスとログは handler.HealthPath を共有し、公開契約のテストは /healthz のリテラルで確認する。

MCP の HTTP エンベロープ、初期化・discover・tools/list・tools/call・未知の操作のディスパッチは固定版 SDK が実装する。自前の NewHandler はその組み立てと共有する利用枠を所有し、newServer は HTTP ごとに Server と定義を生成する。newCollectReviewsHandler が HTTP/SDK の context を連携して collect へ渡す。これらは別の関数のまま同じ handler パッケージに置き、SDK 自体のコードや新しい HTTP controller は追加しない。

reviewmcp の入力、schema、公開用の失敗分類、結果・サイズ変換、埋め込みの分析ガイドは同じ受付に密接な補助処理として handler に置く。非公開定義を保持し、ガイドの版・本文を変更しない。収集手順と利用枠・期限の実行は既存の application、受付ルールと値は domain に維持する。

mcpaccess の HTTP 保護は handler に置き、Access と Host 照合のルールは domain に維持する。Host 拒否、cross-origin 拒否、次の handler という既存の順序を変えない。

## 登録がない経路と対象外

未定義・子パス・旧 /mcp・OAuth metadata の404、healthz の未許可メソッドの405は既存ルーターの動作を使う。対応する handler を新たに作らない。MCP のプロトコルエラーも SDK に委譲する。

設定解析、HTTP サーバーの待ち受け・終了、取得クライアントの構築は cmd/server/adapter に残す。Steam クライアント、CLI、JSON 出力はルーティングの受付ではないため配置を維持し、形式だけの handler フォルダを作らない。テスト用の HTTP サーバー・fixture は本番のルーティング一覧に含めない。

## 配置と動作の検証

internal/handler_architecture_test.go は cmd・internal の本番 Go ファイルを収集し、ServeHTTP メソッド、http.Handler／http.HandlerFunc／mcp.ToolHandler を返す関数、http.HandlerFunc の構築と HandleFunc／AddTool に直接渡す関数を検出する。net/http と MCP SDK の import alias を識別し、検出した自前の受付は adapter/handler・package handler に置くことを確認する。

これは構文上の配置規則の検査であり、動的な型、独自の型エイリアスや任意のルーターの全呼び出し先の解決は行わない。第三者の標準ライブラリ・SDK のソースは対象へ取り込まない。検出例と登録だけを行う処理の許可を入力例でも確認し、既存の TestLayerDependencies と併用する。新たな受付方式を採用する場合は、この一覧と検査範囲も更新する。

公開 HTTP/MCP・エラー・制限・期限・context と資源の契約は既存テストで確認する。移動で入力・期待値を変えず、通常／race 付き結合と同じ本番イメージを検証する。配置と依存の検査の成功だけで実行時の契約を保証したとは扱わない。
