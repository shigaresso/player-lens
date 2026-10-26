# Controller とルーティングの一覧

## 一覧から処理の流れを読む

Issue #77 の配置基準は、handler 直下の本番ファイルを Controller の一覧にすることである。そのファイルで、リクエストの準備・入力変換の呼び出し・UseCase・応答変換までを追う。Issue #80 で自作の Controller の実行メソッド名を Handle に統一した。healthHandler.Handle と collectReviewsHandler.Handle を入口として読む。補助処理を同列に並べず、専用の詳細はサブフォルダ、生成・登録と middleware は外側へ配置する。[ADR-0013](adr/0013-domain-folder-layers.md) に判断を記録する。

| ルーティング・接続元 | Controller または委譲先 | 役割と詳細 |
| --- | --- | --- |
| [RegisterRoutes](../cmd/server/adapter/routes.go) の GET/HEAD /healthz | [handler/health.go](../cmd/server/adapter/handler/health.go) | healthHandler.Handle が開始時刻・応答・結果記録を担当し、標準 http.HandlerFunc が HTTP へ接続する。ログ整形は下位の [logging](../cmd/server/adapter/handler/logging/health_log.go) |
| 同じ RegisterRoutes の /tools | [adapter/server.go](../internal/reviewmcp/adapter/server.go) が構築した SDK の HTTP handler | SDK の生成・共有 limiter・Server 構築を担当する。Controller ではない |
| newServer の collect_reviews 登録と SDK のディスパッチ | [handler/collect_reviews.go](../internal/reviewmcp/adapter/handler/collect_reviews.go) | NewCollectReviewsHandler が固定の依存を組み立て、collectReviewsHandler.Handle が context の準備・入力・UseCase・応答変換・後始末をつなぐ |
| 公開／ローカルの Host/Origin 保護 | [Protect](../internal/mcpaccess/adapter/handler.go)／[ProtectLocal](../internal/mcpaccess/adapter/local.go) | next の手前で検査する middleware。Controller の一覧には含めない |

```text
reviewmcp/adapter/
  server.go                         SDK・共有状態の生成と登録
  collection_definition.go          ツールの定義
  handler/
    collect_reviews.go              Controller：準備→入力→UseCase→応答
    protocol/                      入力・応答・失敗・schema・ガイドの詳細
  internal/testutil/                 テストだけが使う fixture
```

## Controller と詳細の依存

MCP の Controller は Issue #80 で collectReviewsHandler.Handle という名前付きメソッドにまとめた。NewCollectReviewsHandler は HTTP ごとの固定の依存を持つ値を組み立て、その Handle のメソッド値を SDK に渡す。Handle が protocol.ParseInput で入力を生成し、不正時は protocol.InvalidInputResult へ変換する。有効なら application.CollectReviews に同期的な結果処理を渡し、その中で protocol.EncodeResult を呼ぶ。変換が終わるまで同時枠を保持し、HTTP/SDK の context 連携 callback を終了時に解放する。これらの流れを collect_reviews.go に残し、外の処理へ委譲するだけの入口にはしない。

protocol は MCP の入力・結果・公開失敗・schema・埋め込みガイドを担当する。ParseInput、応答生成、InputSchema／OutputSchema と応答量超過のエラーを接続点として公開し、受信型・出力型・分類の詳細は非公開に維持する。構築側のツール定義も公開 schema を使い、上限やガイドの版を二重定義しない。protocol は親の handler と adapter を import しない。

healthz の NewHealthHandler は http.HandlerFunc(handler.Handle) を返す。HTTP の ServeHTTP は標準ライブラリが実装し、応答の実行は healthHandler.Handle が担当する。healthz は HealthPath の1定義を登録とログで共有し、Controller が logging.HealthCheck に固定パス・メソッド・所要時間・書き込み結果を渡す。logging は親を参照せず、JSON 属性・level・安全な原因分類を担当する。受信 query や任意のエラー文字列は出さない。

生成と登録は外側から Controller と下位の詳細を利用し、Controller は専用の詳細と application を利用する。逆向きの依存や、循環を避けるためだけの callback 群は追加しない。既存の domain/application、CLI、外部クライアント、JSON 出力の責務を維持する。

## SDK と middleware の境界

MCP の HTTP エンベロープ・初期化・discover・tools/list・tools/call・未知の操作のディスパッチは固定版 SDK が実装する。newServer は Server の設定とツール登録、collection_definition.go は説明・schema・annotations を担当し、独自の SDK controller は追加しない。

Host/Origin 保護は mcpaccess/adapter に維持し、Host 拒否→cross-origin 保護→next の順序を変えない。未定義・子パス・旧 /mcp・OAuth metadata の404、healthz の未許可メソッドの405は既存ルーターに任せる。http.Handler を返すことだけで Controller と分類しない。Handle はこのアプリの Controller の実行メソッド名であり、Go 全体や SDK が要求する名称ではない。HTTP と MCP の引数・戻り値を統一する共通 interface は追加しない。

## 配置・一覧性・動作の検証

internal/handler_architecture_test.go は cmd・internal の本番 Go ソースを収集し、ServeHTTP の実装、HTTP の名前付き実行関数、MCP ToolHandler の生成、直接登録する callback を構文から検出する。検出した Controller は adapter/handler の直下に置き、直下の本番ファイルには Controller 実装を要求する。純粋な変換・生成だけのファイルを直下に並べず、サブフォルダの詳細と外側の middleware・SDK 構築を許可する。

構文検査は任意の独自ルーターや動的な型、SDK 内部の全到達先を証明しない。Controller の一覧と実際の UseCase 呼び出しは差分レビューと既存の HTTP/MCP テストで確認する。既存のレイヤー依存検査は、本番コードからテスト helper を参照しないことも確認する。

関連テストは担当する場所に置き、共有 fixture は adapter/internal/testutil に限定する。VS Code では Controller と関連テストを同じ親の下にまとめ、詳細フォルダを折りたたむと直下の Controller を探せるようにする。入力・期待値と正常／失敗・資源の契約を維持し、通常・race 付き結合と同じ本番イメージで検証する。
