# 生存確認の結果を構造化ログに記録した

## 背景・目的

[Issue #72](https://github.com/shigaresso/player-lens/issues/72) に対応した。GET/HEAD /healthz の1回の処理結果を記録し、将来の OpenTelemetry 接続へ request context と型付きの属性を引き継げる構成にした。

最新 origin/main（4ec66c112c83f3b756263eabff43cff3ce34a80b）から codex/72-health-structured-logging を作成した。既存ブランチを保持し、upstream は設定していない。push・PR 作成・Issue のクローズは行っていない。

## 変更内容

- main で標準 slog の JSONHandler と service.name を設定し、logger を run と HTTP adapter に渡した。起動と実行失敗の既存アプリケーションログも JSON に揃えた。
- NewHandler はルーティング、healthHandler は応答と計測、非公開の healthCheckLog は1回の結果の属性とレベルを担当する構成にした。logger と時計は呼び出し側で構築し、request context を LogAttrs へ渡した。
- 正常終了を INFO、本文 Write の失敗を ERROR として1回記録した。数値の duration_ms、選択した HTTP ステータス、結果、固定イベント名を持たせた。Write 失敗の原因は安全な分類へ変換し、未加工のエラー文字列、query、ヘッダー、本文を出さない契約にした。
- GET/HEAD の応答、404/405、MCP への委譲を維持した。Write 失敗時の再書き込みを行わず、ログ出力の通常の Write エラーでも応答を変更しないことを確認した。
- HTTP 仕様と ADR-0017 に契約・責務・保存と OTel の未導入範囲を記録した。仕様に残っていた NewSteamFetcher の配置の説明も現在の adapter に合わせた。

## 検証

### 契約ごとの Red・Green

| 契約 | Red | Green |
| --- | --- | --- |
| GET/HEAD の正常な完了ログ | logger と時計を受け取る API が未実装のためコンパイル失敗を確認した | API と通常ログを実装し、応答・JSON1件・固定属性・2.5ms・入力非公開を確認した |
| 本文 Write の失敗ログ | 旧テキスト出力のままで JSON を取得できず失敗した | 分類付きの ERROR を実装し、10入力の原因分類・秘密文字列の非公開・再書き込みなしを確認した |
| 実バイナリの起動失敗ログ | 不正 PORT で旧テキストが stderr に出て JSON として読めず失敗した | 起動側の logger を実装し、非0終了・stdout が空・stderr の JSON1件・既存診断を確認した |

最初の2契約は Docker Compose で `go test -count=1 ./cmd/server/...`、起動失敗は `go test -tags=integration -count=1 ./cmd/server -run '^TestStartupFailureStructuredLog$'` の Green を確認した。

その後、ログの Write エラー時の応答維持、通常／キャンセル済み request context の引き継ぎ、対象外の経路では生存確認ログを出さない契約を順にテストへ追加した。追加時点で Green であり、失敗を作るための実装変更を行っていない。実 HTTP の GET/HEAD と JSON の接続も `go test -tags=integration -race -count=1 ./cmd/server -run '^TestHealthStructuredLogWithHTTP$'` で成功した。

実装と対応するテスト・仕様を 1ae438c、既に Green だった非干渉と接続の回帰テストを 03c372b に分けた。

### 全体検証

- `bash scripts/check-go.sh format-fix` が成功した。別パスの一時コピーに対する wsl_v5 の修正検査は0 issues で、追加の空行修正は不要だった。既存の読み取り専用 Compose 設定は変更していない。
- `bash scripts/check-app.sh` が成功した。gofmt・vet・通常テスト、lint（0 issues）、Git 非依存の3条件、race 付き通常／結合テスト、本番用イメージを検証した。
- 一度ビルドした本番用 image ID `sha256:dc207fd04a1ecef0730aa0041b50b8f976a7f846ec77ecdbbc84bf9255e3f8fe` を使い、HTTP/MCP・Host/Origin・PORT・不正設定・SIGTERM/SIGINT を確認した。
- 実装と回帰テストの各コミットで、必須 pre-commit の Python177テスト・静的検査・index の gofmt と commit-msg が成功した。最終のコミット範囲・差分・Issue/PR 本文の規約検査結果は Issue に記録する。

### AI のセルフレビューと制約

変更対象の差分とテストを確認し、HTTP 応答とログの契約を分けた。ログの status は選択した200であり、クライアントの受信成功を表さない。HEAD も本文なしの応答処理を完了した時点で記録する。panic は既存どおり伝播する。

logger と時計は非 nil かつ並行呼び出し可能なものを渡す契約とし、生成済み依存の再検査や共有グローバル logger の変更を追加していない。既存の期限・終了・接続プールの所有、MCP の取得手順を維持した。新しい依存・共通ログ基盤・資源・goroutine・除外と nolint は追加していない。人によるレビューは未実施。

JSON は stderr に出力する。OpenTelemetry SDK・トレース・メトリクス・転送先・保持期間・保存基盤は未導入であり、将来は起動側の Handler と資源管理を追加する。任意の出力先の非ブロッキングや保存成功、net/http・SDK 自身の全ログの JSON 化は保証しない。

GitHub 認証と Docker は権限付きで実行し、既存の一時 Docker 設定を再利用した。認証設定・必須フック・検査を省略していない。push 前のためコード CI と送信済み head/base による PR 作成検査は未実施で、CI の完了待機も行っていない。実 Steam/LLM/ChatGPT・配置と、coverage 閾値・分岐・mutation の導入・実行は今回の対象外とした。

## 参照仕様・ADR

- docs/specs/http-server.md「HTTP の契約」「生存確認の構造化ログ（Issue #72）」「設定と終了の契約」「検証との対応」
- docs/specs/review-mcp.md の既存ルーティングと資源の契約
- ADR-0001 の実行環境、ADR-0002 の検査手順、ADR-0003 の固定版ツールと空行規則、ADR-0013 の起動部分と領域内 adapter の境界、ADR-0017 のログ設計
- Go 公式 log/slog、OpenTelemetry の otelslog bridge とログデータモデル

## 関連 Issue

Closes #72
