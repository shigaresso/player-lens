# ADR-0013：各領域フォルダの内部をレイヤーで分類する

## 状態と対象

採用。[Issue #56](https://github.com/shigaresso/player-lens/issues/56) に対応する。既存の steam・reviewmcp・reviewcli・jsonoutput・mcpaccess と2つの cmd を維持し、それぞれの内部に domain・application・adapter を配置する。ADR-0005・0006 の配置を部分置換し、業務・保存・数値型・外部形式・資源の契約は維持する。

## 背景と比較

ユーザーの意図は、各領域フォルダの内部を domain・application・adapter のレイヤーで分類することである。複数の領域を internal/review へ統合する案と、レイヤーを model・handler・config などの役割名で置き換える案は採用しない。

実在する責務のあるレイヤーだけを作る。委譲だけの互換関数、型エイリアス、空のレイヤー、将来用の依存は追加しない。

## 決定と依存

| 既存フォルダ | domain | application | adapter |
| --- | --- | --- | --- |
| internal/steam | ID・API 数値型・レビュー・条件・結果・正規化と収集状態 | 収集・収集保存と利用側 interface | HTTP クライアント・要求と応答の変換 |
| internal/reviewmcp | 上限と受付・返却ルール | 8秒の期限・利用枠と収集の実行 | HTTP/SDK・入力・schema・結果・分析ガイド |
| internal/reviewcli | 追加しない | 追加しない | 引数の解析と収集保存への接続 |
| internal/jsonoutput | 追加しない | 追加しない | JSON 変換と writer 出力 |
| internal/mcpaccess | 公開 URL・Host の設定と照合 | 追加しない | HTTP の Host/Origin 保護 |
| cmd/server | 追加しない | 追加しない | 設定の解析・HTTP・取得依存の構築 |
| cmd/collect-reviews | 追加しない | 追加しない | CLI の表示と終了コード |

steam/application は steam/domain と標準ライブラリに依存し、具体的な adapter を知らない。公式 API 型と ReviewType は同じ domain へ移し、ADR-0012 の基底型・定数の定義元・値を維持する。

reviewmcp/application は reviewmcp/domain と steam の application/domain を使い、SDK の型を含めない。実行結果は収集結果・拒否理由・元の失敗を保持する。入力の不正は adapter が先に拒否し、利用枠を消費しない。application が取得開始時に枠を確保し、defer で返却する。開始回数は戻さず、開始前キャンセルは枠を消費しない。adapter は既存の結果とエラーへ変換し、公開 schema を維持する。

CLI と JSON 出力の責務は adapter であり、その内部へ業務操作を複製しない。Issue #82 で mcpaccess/domain の Access はローカル用・公開用の生成済み条件、モード照会と Host の照合を提供する。adapter.Protect は設定と next を受け取り、nil・ゼロ値・Host・Origin の既存の拒否を維持する。内部の authority 文字列表現は公開しない。

cmd は main と起動の組み立てを root に維持する。既存の I/O 処理を内部の adapter へ移し、listener・接続・signal・goroutine・context の終了担当を変えない。

### Controller を探す場所（Issue #77）

handler 直下の本番ファイルは、リクエストの事前処理・UseCase 呼び出し・事後処理を読む Controller の一覧とする。HTTP/MCP の生成関数や middleware、すべての関連処理を同列に配置する規則は採用しない。

healthz は cmd/server/adapter/handler/health.go、MCP の context 連携と収集操作への接続は reviewmcp/adapter/handler/collect_reviews.go に置く。専用の入力・応答・schema・ガイドは handler/protocol、healthz のログ整形は handler/logging に置き、直下の Controller から呼ぶ。単に入口を委譲するだけのファイルへ流れを隠さない。

Issue #80 で自作の Controller の実行メソッド名を Handle に統一する。healthHandler.Handle は標準 http.HandlerFunc、collectReviewsHandler.Handle は SDK の ToolHandler に適合するメソッド値として生成関数から返す。HTTP と MCP の引数・戻り値は各接続の契約を維持し、共通 interface や独自の委譲メソッドは追加しない。ServeHTTP は HTTP の契約名であり、Controller 一般の名称とは扱わない。MCP は HTTP ごとの固定の依存と、Handle が所有する呼び出しごとの context を分け、HTTP の受付は SDK に任せる。

MCP の SDK と共有 limiter の構築・Server の設定と登録・ツール定義は外側の adapter、Host/Origin の middleware は mcpaccess/adapter に配置する。生成と登録、UseCase と表現の既存の責務・寿命を維持する。補助パッケージは親の Controller や構築側を参照しない。必要な入出力・schema の API だけを公開し、非公開の受信型や JSON 出力型を一律に公開しない。

登録元・Controller・補助処理と SDK 委譲の対応は [受付とルーティングの一覧](../handler-routing.md) に記録する。配置検査は Controller と直下の一覧性を確認し、生成・middleware の外側配置と専用の補助フォルダを許可する。構文による検査であり、任意の動的な型や SDK 内部の全ディスパッチを証明するものではない。既存のレイヤー依存・本番からのテスト helper 参照の禁止も維持する。

## 検証と制約

テストと埋め込みガイドも移動先へ置く。分類の都合で既存の期待値を変更せず、通常・race 付き結合・本番用イメージで確認する。レイヤー間の import と本番からのテスト helper 依存を検査する。Go の公開範囲を変える必要がある定義には契約コメントを付ける。

依存検査は internal/architecture_test.go で cmd・internal の全 Go ファイルを収集する。各 domain/application が依存できる領域型・利用側契約だけを許可し、adapter・SDK と分類前のパッケージへの依存を検出する。testing を import する本番ファイルのパッケージは共有テスト helper として扱い、他の本番コードからの参照を拒否する。

パッケージと import は増える。層を作っただけで理解・動作を保証したとは扱わず、操作一覧と実際の依存で責務を示す。過去の作業記録のパスは当時の記録として保持する。

## 参照

- [ユースケース](../use-cases.md)、[開発手順](../development.md)
- ADR-0004〜0009・0011・0012。実施結果は Issue #56 の作業記録に残す。

収集状態の生成・ページ反映と application の取得手順の境界は [ADR-0014](0014-steam-collection-state.md) で具体化する。

利用枠は応答変換の完了まで保持する。Issue #62 で application に同期的な結果処理関数を渡し、adapter が変換を行う間も同じ defer の範囲に含める。[ADR-0016](0016-mcp-response-slot-lifetime.md) に契約を具体化する。
