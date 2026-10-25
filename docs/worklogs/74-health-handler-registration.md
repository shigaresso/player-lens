# healthz の handler 生成と経路登録を分離した

## 背景・目的

[Issue #74](https://github.com/shigaresso/player-lens/issues/74) に対応した。healthz を他のエンドポイントの見本として読めるよう、生成は New、既存の対象への登録は Register という名前と責務に揃えた。

PR #73 が統合済みの最新 origin/main（07e0da74b6b625923eb4bccc3addff7b4422e5b0）から codex/74-health-handler-registration を作成した。既存ブランチを保持し、upstream は設定していない。push・PR 作成・Issue のクローズは行っていない。

## 変更内容

- NewHealthHandler は logger と時計を受け取り、生存確認の handler を生成する構成にした。生成時に時刻取得・ログ出力を行わず、応答と計測の実装は維持した。
- RegisterRoutes は既存の http.ServeMux に生成済みの health/MCP handler を登録する処理にした。ルーター・handler・logger・時計を生成せず、登録時に handler を呼ばない契約にした。MCP が nil の場合の未登録、標準 ServeMux の競合時の契約を維持した。
- run が両方の handler とルーターを組み立て、登録してから listener を生成し、Serve へ渡す構成にした。旧 server adapter の NewHandler は削除し、互換ラッパーを残さず呼び出し元を移行した。reviewmcp.NewHandler は維持した。
- 登録テストに logger・時計を渡さず、任意の生成済み handler を使えることと、無関係な既存経路の保持を確認した。旧 handler_test.go は health_test.go に改名し、既存の入力・期待値を引き継いだ。
- HTTP 仕様に起動時とリクエスト時の流れを記録し、ADR-0017 と文書一覧の生成・登録の説明を更新した。新しいレイヤー・依存・汎用 middleware は追加していない。

## 検証

### Red・Green と既存契約の移行

| 契約 | Red | Green |
| --- | --- | --- |
| 既存ルーターへの経路登録 | TestRegisterRoutes を追加し、RegisterRoutes 未実装によるコンパイル失敗を確認した | 実装後、10状況で指定 handler への委譲・登録時の非実行・既存経路の保持・MCP 未指定・404/405 と Allow を確認した |
| health handler の生成 | TestNewHealthHandlerDoesNotProcessRequest を追加し、NewHealthHandler 未実装によるコンパイル失敗を確認した | 実装後、非 nil の handler を返し、生成時の時計呼び出しとログ出力がないことを確認した |

各段階で固定版 Docker Compose の verify サービスを使用し、`go test -count=1 ./cmd/server/adapter -run '^TestRegisterRoutes$'` と `go test -count=1 ./cmd/server/adapter -run '^TestNewHealthHandlerDoesNotProcessRequest$'` の Red・Green を順に確認した。

その後の呼び出し元の移行と旧 API 削除は既存の振る舞いを変えない Refactor として扱い、形式だけの Red を追加していない。`go test -tags=integration -race -count=1 ./cmd/server/...` が成功した。既存の GET/HEAD、書き込み失敗の分類、ログ出力失敗、request context、対象外経路、実 HTTP と JSON、MCP と終了処理の入力・期待値を維持した。

生成と登録の分離、対応するテスト・呼び出し元・仕様を b401ed7 にまとめ、検証記録は別の文書コミットに分けた。実装コミットの必須フックで Python177テスト・静的検査・index の gofmt・commit-msg が成功した。

### 全体検証

- `bash scripts/check-go.sh format-fix` が成功した。
- `bash scripts/check-app.sh` が成功した。gofmt・vet・通常テスト、lint（0 issues）、Git 非依存3条件、race 付き通常／結合テストを確認した。
- 一度ビルドした本番用 image ID `sha256:bd560304b11c12ff90593bfb53d7fae0cfa612f1d76f5e5b40ddaddbf8ccd3c3` を使い、HTTP/MCP・Host/Origin・PORT・不正設定・SIGTERM/SIGINT を確認した。
- 基点と比較し、healthHandler.ServeHTTP と health_log.go の実装が同一であることを確認した。差分の空白検査も成功した。
- 必須フック、最終のコミット範囲・Issue/PR 本文の規約検査とローカルコミットの結果は Issue に記録する。

### AI のセルフレビューと制約

生成と登録が起動部分から順に読めることと、リクエストごとに組み立てを繰り返さないことを確認した。登録前に handler を生成し、listener を作成する前に登録する構成にした。listener の所有を受け取る Serve、接続プールの defer、context と goroutine の終了処理は変更していない。

登録する mux と health handler、生成に渡す logger と時計は有効な依存を呼び出し側が渡す契約とし、再検査や独自の panic 回復を追加していない。登録済みの競合するパターンは呼び出し側が避ける。登録の可変状態は起動時に設定し、受付中に変更しない。

ログを検証しないテストは io.Discard、検証するテストはメモリ上の記録または子プロセスの出力捕捉を維持した。ログ属性・標準エラー出力・保存と OTel の未導入範囲を変更していない。人によるレビューは未実施。

GitHub 認証と Docker は権限付きで実行し、既存の一時 Docker 設定を再利用した。必須フックと検査の省略、読み取り専用 Compose 設定・ツール版・除外条件の変更は行っていない。push 前のため pre-push とコード CI、送信済み head/base による PR 作成検査は未実施。実 Render 配置・実外部サービス接続、coverage 閾値・分岐・mutation（Stryker を含む）の導入・実行は対象外。

## 参照仕様・ADR

- docs/specs/http-server.md「HTTP の契約」「起動時の生成と経路登録（Issue #74）」「生存確認の構造化ログ（Issue #72）」「設定と終了の契約」「検証との対応」
- ADR-0013 の cmd の組み立てと adapter の境界、ADR-0017 の生成・登録・応答・計測・ログの責務
- docs/development.md の各フック・アプリケーション検査・Issue/PR 作成手順、ADR-0001・0002・0003 の既存実行環境と検査

## 関連 Issue

Closes #74
