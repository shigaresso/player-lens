# Controller の実行入口を Handle に統一した

## 背景・目的

[Issue #80](https://github.com/shigaresso/player-lens/issues/80) に対応した。[PR #79](https://github.com/shigaresso/player-lens/pull/79) の統合を確認し、最新 origin/main の 748230ee662040287b6b4dd5848337139a70890a から codex/80-named-mcp-controller を作成した。既存ブランチとコミットを保持し、upstream は設定していない。

Controller の実行入口を healthHandler.Handle と collectReviewsHandler.Handle に統一した。最初の実装では MCP のみを名前付きメソッドにし、healthz を ServeHTTP のままにしていたため、ユーザーが求めた統一を満たしていなかった。Issue の範囲と完了条件を訂正し、既存のローカルコミットを保持して追加で対応した。振る舞いの追加や既存の無名関数の不具合修正とは扱わない。

## 変更内容

- healthz の実行メソッドを healthHandler.Handle に改名し、NewHealthHandler は http.HandlerFunc(handler.Handle) を返す形にした。HTTP の ServeHTTP は標準ライブラリが実装し、自作の Controller の実行は Handle から読む。応答・計測・ログの処理本体と、既存の公開 API・テストは変更していない。
- handler/collect_reviews.go に非公開の collectReviewsHandler を定義し、固定の fetcher・HTTP リクエストの context・共有 limiter を保持した。NewCollectReviewsHandler はこの値を組み立て、SDK の ToolHandler に適合する Handle のメソッド値を返す。既存の公開関数のシグネチャを維持し、公開型・interface・互換ラッパーを追加していない。
- Handle に context の準備、入力変換、不正時の応答、UseCase、同期的な応答変換、defer による後始末をまとめ、従来の collect を削除した。実行時に receiver のフィールドを書き換えず、呼び出しごとの context と結果はローカル変数にした。
- SDK の生成・設定・ツール登録は外側に維持した。入力・応答の詳細は protocol、期限・利用枠は application/domain に残した。HTTP の ServeHTTP は SDK の担当として維持し、独自の HTTP Controller は追加していない。
- 既存テストの collect 呼び出し20箇所を、テスト専用の callCollectReviews から公開する SDK 受付へ接続した。共通 helper は t.Helper を使用し、SDK の request 構築だけをまとめた。本番の互換関数は残していない。
- [受付一覧](../handler-routing.md)、[MCP 仕様](../specs/review-mcp.md)、[操作一覧](../use-cases.md)、[ADR-0013](../adr/0013-domain-folder-layers.md) に最終の呼び出し経路と所有範囲を記録した。

## 検証

### Green からの回帰テストと Refactor

Docker Compose の verify で、変更前に `go test -count=1 ./internal/reviewmcp/adapter/... ./internal` が成功した。

Controller 経由の panic の後始末を確認する TestCollectReviewsHandlerPanicCleanup をテストだけで追加した。取得中に意図した panic を発生させ、同じ原因が伝わること、HTTP/SDK の callback 登録が解除されること、所有 context がキャンセルされ親は維持されること、同時枠が返却されることを確認した。`go test -count=1 ./internal/reviewmcp/adapter/handler -run '^TestCollectReviewsHandler'` は既存実装でも Green だったため、形式だけの Red を作らず、テストのみのコミットにした。

名前付きメソッドへの整理とテストの機械的な接続更新後に、同じ関連パッケージのコマンドで Green を確認した。既存テストの入力・期待値は変更していない。既存の配置検査は ToolHandler を返す生成関数から今回のファイルを検出できるため、規則や検査対象を変更せず成功を確認した。無名関数という形式自体を拒否する検査は追加していない。

入口名の統一の訂正前・訂正後にも、`go test -count=1 ./cmd/server/... ./internal/reviewmcp/adapter/... ./internal` の Green を確認した。healthz は処理本体とテストを維持し、生成側の標準 http.HandlerFunc と Handle のメソッド値の接続を変更した。配置検査は名前に依存しない HTTP のシグネチャも検出するため、規則を変更せず成功した。

### 全体検証と差分確認

- `bash scripts/check-go.sh format-fix` が成功した。
- `bash scripts/check-app.sh` が成功した。gofmt・vet・全通常テスト、lint 0 issues、Git 非依存3条件、race 付き通常／結合テストを実行した。結合テストだけの結果とは扱わない。
- 統一後の同じ本番 image ID `sha256:97d98e1d88e565a0bdc32e75d20ce285dc635e46e638e09380479f74791054cd` で HTTP/MCP・Host/Origin・PORT・不正設定・SIGTERM/SIGINT と正常終了を確認した。
- 回帰テストのコミットと比較し、既存テストの変更が受付呼び出し20箇所と helper・必要な import だけで、残りの本文が一致することを照合した。application/domain、SDK の構築、protocol、依存の版には差分がないことを確認した。
- テスト・MCP の Refactor・入口名の統一のコミットで必須フックが成功した。文書コミットのフックと、最終の差分・コミット範囲・本文・リンクの検査結果は Issue に記録した。

### AI のセルフレビューと制約

healthz と MCP の本番の実行メソッドが両方とも Handle であることと、HTTP と MCP で引数・戻り値が異なることを確認した。Handle はアプリ内の目印であり、Go 全体の命名規則とは説明しない。HTTP ごとの固定の依存と SDK の実行ごとの変数を区別し、共有可変 context を追加していないことを確認した。SDK の HTTP リクエストごとの Server 構築、limiter の共有と開始回数、8秒の期限、同期 callback による応答変換までの枠保持、既存のエラー原因と公開応答を維持した。callback の解除・所有 context のキャンセル・枠の返却の順序を既存と照合した。人によるレビューは未実施。

GitHub 認証と Docker は権限付きで実行し、検証用の一時 Docker 設定を再利用した。必須フックを省略していない。push・PR 作成・Issue のクローズは行っていない。pre-push・コード CI、実 Render 配置・実外部サービス接続は未実施。coverage 閾値・分岐・mutation（Stryker を含む）は導入・実行していない。

## 参照仕様・ADR

- [HTTP 仕様](../specs/http-server.md) の生存確認・生成と登録・構造化ログ、および [ADR-0017](../adr/0017-health-structured-logging.md)
- [MCP 仕様](../specs/review-mcp.md) の「ツール登録と実行の境界」「結果変換の境界」「結果・失敗・資源」
- [受付一覧](../handler-routing.md) の Controller と詳細の依存、[ADR-0013](../adr/0013-domain-folder-layers.md) の Controller の配置と依存
- [ADR-0007](../adr/0007-review-mcp.md) の SDK と HTTP ごとの構築、[ADR-0008](../adr/0008-mcp-usage-limits.md)・[ADR-0016](../adr/0016-mcp-response-slot-lifetime.md) の共有状態と結果処理完了までの枠保持・返却
- [開発手順](../development.md) の Docker・テスト・必須フックと、関連する既存の実行・整形・検査方針

## 関連 Issue

Closes #80
