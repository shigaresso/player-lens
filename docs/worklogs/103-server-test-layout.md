# サーバー起動処理と関連テストの配置・表示を整理した

## 背景・目的

[Issue #103](https://github.com/shigaresso/player-lens/issues/103) に対応した。PR #102 統合済みの最新 origin/main（61e9cb2）から codex/103-server-test-layout を作成した。既存ブランチと履歴を保持し、upstream は設定していない。

## テストの統合

cmd/server/adapter の mcp_handler_test.go は RegisterRoutes を検証していたため、固有の4ケースを routes_test.go の TestRegisterRoutes のテーブルへ引き継いだ。既存ケースとの重複を整理し、MCP handler へ渡すリクエストのパスの確認も残した。旧ファイルは削除した。

旧 TestMCPRoute の8ケースは次の検査へ引き継いだ。

| 入力と期待結果 | 引き継ぎ先 |
| --- | --- |
| POST /tools は202と MCP への委譲 | TestRegisterRoutes の既存ケース。渡したパスも確認する |
| GET /tools は202と MCP への委譲 | TestRegisterRoutes の既存ケース。渡したパスも確認する |
| POST /tools/child は404と非委譲 | TestRegisterRoutes の既存ケース |
| POST /tools/ は404と非委譲 | TestRegisterRoutes へ移したケース |
| POST /mcp は404と非委譲 | TestRegisterRoutes へ移したケース |
| GET の OAuth metadata は404と非委譲 | TestRegisterRoutes へ移したケース |
| POST の OAuth metadata は404と非委譲 | TestRegisterRoutes へ移したケース |
| GET /healthz は200と MCP への非委譲 | TestHealthRoutes の実 health handler の200・本文・ヘッダーと、TestRegisterRoutes の登録先・呼び出し回数の確認 |

この変更だけを a69f471 にコミットした。

internal/reviewmcp/adapter の TestNewHandlerRejectsInvalidLimits は、同じパッケージの server_test.go へ移した。旧 limits_test.go は削除した。origin/main の関数本文と比較し、入力・nil handler と ErrInvalidArgument の検証が文字列として同一であることを確認した。この移動だけを c3052dd にコミットした。

## ネスティングと実行条件

.vscode/settings.json の server.go の子から limits_test.go を、handler.go の子から削除済みの mcp_handler_test.go を外した。cmd/server/adapter/limits_test.go は同名の limits.go に対応し、MCP adapter の生成契約と混同しない設定にした。通常・integration テストを同名の実装へまとめる汎用パターンは維持した。

JSON の構文と、対象フォルダの設定上の親子対応7組を確認した。cmd/server/adapter の routes・limits・port_number・mcp_config・server の通常テストと steam_fetcher の integration テスト、internal/reviewmcp/adapter の server の通常テストが、それぞれ対応する実装だけを親候補に持つことを確認した。これは設定の静的確認であり、エディター上の実表示の確認ではない。

初回のテスト整理では本番 Go コード・結合テストのファイルと build tag は変更していない。steam_fetcher_integration_test.go はローカル TLS の実接続と接続解放を検証するため、独立ファイルと integration 指定を維持した。表示上のネスティングはテストの実行対象を変えない。

## 追加対応：run のファイル分離

ユーザーの追加依頼に基づいて同じ Issue の範囲と完了条件を更新し、cmd/server/main.go から run と shutdownTimeout を run_server.go へ移した。main.go はプロセスの入口と run の呼び出しを維持する。同じ package main の中でファイルを分け、既存の run_test.go は run_server_test.go へ改名した。

追加対応前の HEAD と比較し、main と run の関数本文・shutdownTimeout の定義・テストファイル全体が同一であることを確認した。import は各ファイルで使うものへ分けた。設定の読み取り・依存構築・defer・待ち受け・ログ・終了の順序は変更していない。

専用のネスティング設定は追加せず、既存の汎用パターンで run_server_test.go の親候補が run_server.go だけになることを確認した。これは設定の静的確認であり、エディター上の実表示の確認ではない。

追加対応前の Green は直前の全体検証結果を使い、形式的な Red は作らなかった。移動後に Docker Compose verify で go test -count=1 ./cmd/server を実行し、通常テストの成功を確認した。

追加対応後にも bash scripts/check-app.sh が終了コード0で成功した。整形・vet・全通常テスト・lint 0 issues・Git 非依存3条件・race 付き通常／結合テストを確認した。本番用 image ID sha256:07003c543490fb2e36cb429bd563b4bd07ba06c2ff5f7f4bedd8aae5e9cfbb8d を一度ビルドし、同じイメージで HTTP/MCP・PORT・不正設定の非0終了・SIGTERM/SIGINT の正常終了を確認した。main.go からのファイル分離後も、パッケージ単位のビルドと既存の起動・終了の契約が成功した。

## 初回のテスト整理の検証

テストの構成整理なので形式的な Red は作らず、変更前後の Green と契約の引き継ぎを確認した。いずれも固定版 Docker Compose verify で -count=1 を指定した。

- 変更前：go test -count=1 ./cmd/server/adapter ./internal/reviewmcp/adapter -run '^(TestRegisterRoutes|TestHealthRoutes|TestMCPRoute|TestNewHandlerRejects.*)$' が成功した。
- ルーティング統合後：go test -count=1 ./cmd/server/adapter -run '^(TestRegisterRoutes|TestHealthRoutes|TestOtherRoutesDoNotEmitHealthLog)$' が成功した。
- 生成契約の移動後：go test -count=1 ./internal/reviewmcp/adapter -run '^TestNewHandlerRejects' が成功した。
- テスト整理の両コミットで、pre-commit のステージ済み秘密情報・Python 182テスト・静的検査・index 整形と、commit-msg の秘密情報・規約検査が成功した。

### 全体検証

bash scripts/check-app.sh が終了コード0で成功した。

- gofmt の差分検査、go vet ./...、go test -count=1 ./... が成功した。
- 固定版 golangci-lint は通常と integration タグのコードを検査し、0 issuesだった。
- Git の失敗・不在・checkout の不在という3条件で、Go の実行と子ビルドが成功した。
- go test -tags=integration -race -count=1 ./... が成功した。通常テストと結合テストの両方を含む。既存の HTTP/MCP・接続解放・起動失敗・終了処理を確認した。
- 本番用 image ID sha256:91ef6796cc62a12379761e6d7bc759e51240034db9b56aa010e23608c9009e16 を一度ビルドし、同じイメージで非 root 設定、ソース共有なしの起動、HTTP/MCP・ローカル／公開 Host・Origin、PORT、不正設定の非0終了、SIGTERM／SIGINT の正常終了を確認した。実 Steam は呼んでいない。
- 変更文書のローカルリンク24件の参照先が存在し、端末のホームパスが変更内容に含まれないことを確認した。リンクのアンカーは対象外とした。

### AI のセルフレビューと未実施項目

旧ルーティングテストの8ケースを上記の引き継ぎ表で照合した。移動した固有ケースは、状態コードだけでなく handler が実行されないことも確認する。重複ケースの集約では、既存の呼び出し回数・登録先・実 health handler の本文とヘッダーを確認するテストを維持した。未初期化 Limits のテストは入力と検証本文が同一であることを確認した。人によるレビューは未実施。

Go の実行と必須フックの Docker は、前回の作業で用意した認証情報のない一時 DOCKER_CONFIG と既存の Docker ソケットを使った。GitHub 認証・Docker・Git の書き込みは権限付きで実行し、検査対象・設定・フックを省略・緩和していない。一時ログ・PR 用本文はリポジトリ外に置き、Git・Docker のビルドコンテキストに含めていない。

push・PR 作成・Issue のクローズは行っていない。pre-push・送信済み head/base の PR 作成検査・コード CI は未実施。エディター上の実表示・実 Steam／LLM 接続・Render 配置は未確認／対象外とした。coverage 閾値・分岐・mutation（Stryker を含む）は現行では未導入であり、今回の結果を品質目標の100%達成とは扱わない。

## 参照仕様・ADR

- [開発手順](../development.md) の「VS Code のファイルネスティング」「各フックの責務」「Go・Compose・本番用イメージの検査」
- [HTTP サーバー仕様](../specs/http-server.md) の「HTTP の契約」「検証との対応」、[MCP 利用制限仕様](../specs/mcp-usage-limits.md) の「設定値の生成と検証」
- [ADR-0013](../adr/0013-domain-folder-layers.md) の「Controller を探す場所」「検証と制約」、ADR-0001・0002・0003 の既存の実行と検証の方針

## 関連 Issue

Closes #103
