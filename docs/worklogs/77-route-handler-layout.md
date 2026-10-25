# アプリ全体の受付処理を handler 配下へ統一した

## 背景・目的

[Issue #77](https://github.com/shigaresso/player-lens/issues/77) に対応した。ルーティングから割り当てられる自前の処理を、各領域の adapter/handler で見つけられる構成に揃えた。healthz に加え、MCP HTTP/SDK とツール実行、Host/Origin 保護を対象とした。

前提の PR #78 が統合済みの最新 origin/main（264b96c7e0621341175a2a74b914514b0a9a6ba2）から codex/77-route-handler-layout を作成した。既存ブランチを保持し、upstream は設定していない。push・PR 作成・Issue のクローズは行っていない。

## 変更内容

| 領域 | 移動内容 | ファイル数 |
| --- | --- | --- |
| cmd/server/adapter/handler | healthz の処理・ログと直接の契約テスト | 5 |
| internal/reviewmcp/adapter/handler | SDK の組み立て・ツール受付・入力・schema・結果・ガイドとテスト | 39 |
| internal/mcpaccess/adapter/handler | 公開／ローカルの Host/Origin 保護とテスト | 4 |

- HTTP の RegisterRoutes は外側の adapter に残し、生成済み handler を登録する契約を維持した。共有パスだけを handler.HealthPath として公開し、登録とログが1定義を参照する構成を引き継いだ。
- MCP の newServer とツール定義は構築の補助、newCollectReviewsHandler と collect は実行入口として既存の関数分離を維持した。密接な入力・結果・schema とガイドを同じパッケージに置き、非公開の補助処理を公開したり互換ラッパーを追加したりしていない。
- ルーターとの接続を確認する TestOtherRoutesDoNotEmitHealthLog は外側の routes_test.go に置き、直接の応答・ログの契約は handler 配下に維持した。呼び出し元・通常／結合テストの import と package 宣言、埋め込みガイドを読む本番イメージ検査とその fixture のパスを機械的に更新した。
- [受付とルーティングの一覧](../handler-routing.md) に全入口・登録元・SDK 委譲・補助処理と配置検査の範囲を記録し、実装へのリンクを付けた。ADR-0013・0017・0009、HTTP/MCP・公開保護・分析ガイドの仕様、操作一覧とガイド本文へのリンクを更新した。
- 新しい配置検査を追加し、既存の TestLayerDependencies と併用した。domain/application、CLI、外部 API クライアント、JSON 出力の配置は維持した。ほかの役割フォルダや空のパッケージ、依存は追加していない。VS Code のネスティングはファイル名による既存設定のまま使えるため変更していない。

## 検証

### 配置規則の Red・Green

internal/handler_architecture_test.go の TestHandlerPlacement と、構文による検出・許可の12入力例を追加した。Docker Compose verify で `go test -count=1 ./internal -run '^TestHandlerPlacement'` を実行し、旧配置の health.go、mcpaccess の handler.go・local.go、reviewmcp の handler.go・collection_handler.go の5ファイルが handler 外にあることを検出する Red を確認した。

3領域と呼び出し元を移動した後、同じコマンドが Green になった。検査と期待結果を緩めていない。自前の HTTP の型・関数・構築と MCP の ToolHandler、直接登録する関数を対象とし、登録だけの処理・SDK の Server 構築 callback・外向きクライアントは入力例で許可を確認した。動的な型や任意の独自ルーターを完全に解析する検査とは扱わない。

移動に関しては既存の正常・失敗・副作用の入力と期待値を引き継ぎ、形式だけの Red を追加していない。関連する `go test -tags=integration -race -count=1 ./cmd/server/... ./internal/mcpaccess/... ./internal/reviewmcp/... ./internal` が成功した。

配置検査とその規則に対応する移動・呼び出し元・仕様を c3d40ee にまとめ、検証記録は別の文書コミットに分けた。

### 全体検証と内容保持

- `bash scripts/check-go.sh format-fix` が成功した。
- `bash scripts/check-app.sh` が成功した。gofmt・vet・通常テスト、lint（0 issues）、Git 非依存3条件、race 付き通常／結合テストを確認した。
- 一度ビルドした本番用 image ID `sha256:892751a273f0802204a1e242a8283df2566821135b29597c817639cd647fffe6` を使い、HTTP/MCP・Host/Origin・PORT・不正設定・SIGTERM/SIGINT を確認した。本番イメージにソースを共有せず、移動したガイドの埋め込みを既存の検査で確認した。
- 移動した48ファイルを基点と比較し、package・import・alias・HealthPath の共有と、必要な接続テストの再配置以外の本文が一致することを確認した。ガイド本文と版も維持した。移した接続テストは生成関数の参照を除いた本体が一致した。
- 実装コミットで必須フックの Python177テスト・静的検査・index の gofmt・commit-msg が成功した。最終のコミット範囲・差分・Issue/PR 本文と文書コミットの検査結果は Issue に記録する。

### AI のセルフレビューと制約

全登録元と処理入口、受付保護、SDK が実装するプロトコルのディスパッチを確認した。MCP の limiter を生成する位置と共有期間、結果変換まで同時枠を保持する呼び出し順、HTTP/SDK の context 連携と callback の解除、エラー原因と公開結果を維持した。移動で登録ごと・呼び出しごとの新しい利用枠を作っていない。

HTTP の listener・接続・終了待機と取得クライアントの解放は既存の担当に維持した。ログを検証しないテストの io.Discard、検証するテストのメモリ出力と子プロセスの捕捉も引き継いだ。新しい controller や SDK の処理の複製、不要な公開型は追加していない。人によるレビューは未実施。

GitHub 認証と Docker は権限付きで実行し、既存の一時 Docker 設定を再利用した。必須フック・検査を省略せず、読み取り専用 Compose 設定・ツール版・除外条件を変更していない。push 前のため pre-push・コード CI と送信済み head/base による PR 作成検査は未実施。実 Render 配置・実外部サービス接続、coverage 閾値・分岐・mutation（Stryker を含む）の導入・実行は対象外。

## 参照仕様・ADR

- docs/handler-routing.md の全登録元・割り当て先・補助処理と配置検査、ADR-0013 の領域とレイヤーを維持した handler 配置
- docs/specs/http-server.md の HTTP・構造化ログ・生成と登録・終了・検証、ADR-0017
- docs/specs/review-mcp.md の接続・入力・ツール登録と実行・結果・失敗・資源、docs/use-cases.md の操作境界
- docs/specs/mcp-public-access.md・ADR-0009 の受付保護、ADR-0008・0016 の利用枠と応答変換までの寿命、docs/specs/review-analysis.md・ADR-0011 の埋め込みガイドの契約
- docs/development.md と ADR-0001・0002・0003 の既存の実行・検査手順

## 関連 Issue

Closes #77
