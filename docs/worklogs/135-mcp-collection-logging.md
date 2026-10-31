# Issue #135：MCP のレビュー収集の構造化ログ

## 範囲と責務

最新取得時の origin/main（2ab0197d84b57aefece397ce8d4c233bc772229d）から codex/135-mcp-collection-logging を作成した。push・PR 作成・Issue のクローズは行わない。

main の JSON/stderr logger と計測時計を run・NewHandler・newServer から Controller へ渡した。NewHandler で logger・時計の nil を拒否し、構築中には計測・ログを実行しない。既存 limiter の時計と回数の規則は変更していない。

Controller が開始時刻と入力・収集・応答生成の結果を持ち、handler/logging の ReviewCollection が安全な属性・レベル・分類を決める。通常 return と panic にそれぞれ1回記録し、context の解放より先に処理中の context を渡す。panic は値を転載せず、回復せずに伝播させる。application・domain にログ依存を追加していない。

公開コードは既存の protocol の分類から取得し、エラー文章や応答 JSON を読み戻さない。ログでは入力・エラー全文・URL・レビュー本文・カーソルを出さず、固定の安全な分類のみを記録する。収集件数は本文省略前の件数とし、送信完了や全件取得を意味させない。収集結果がない入力拒否・受付拒否には件数を付けない。

実ログの新しいファイル・保存先・buffer・goroutine は追加せず、出力先を Close しない。Git/Docker に含む実データと ignore の対象追加はない。通常のテストには明示的な破棄 logger を渡し、契約テストはメモリ上の出力先を使う。

## 参照した仕様と ADR

- docs/specs/review-mcp.md：入力・結果・失敗・資源、入力原因の保持、結果変換。今回の属性・省略条件・レベル・責務を追加した。
- docs/specs/http-server.md と ADR-0017：既存の healthz の slog・計測・context・保存の境界。
- docs/specs/data-and-research.md：現段階の収集と未導入範囲。
- docs/handler-routing.md、ADR-0013：Controller の一覧性と詳細フォルダ。MCP の logging の配置を追記した。
- ADR-0007・ADR-0020：SDK・公開応答と枠返却・context の寿命。
- ADR-0021：今回の観測単位・安全なログ・将来の OTel 接続の決定。docs/README.md に参照を追加した。
- docs/development.md：検証環境・必須フック・秘密情報と Issue 作成手順。

## 契約ごとの Red・Green

1. 安全なログ型：TestReviewCollectionWrite を先に追加し、ReviewCollection が未定義であるコンパイル失敗を確認した。実装後に logging と既存 protocol のテストが Green となった。成功、JSON の構文・型、条件不正、3種類の受付拒否、キャンセル、期限、HTTP ステータス、カーソル循環、Steam 応答の上限・形式、通信タイムアウト、未知原因、本文省略、応答生成不能、panic を固定属性・レベルへ対応させた。結果未設定の防御的な分類も回帰確認した。
2. Controller の接続：計測・回数・処理中 context と値・writer 失敗の非干渉のテストを先に追加し、logger と時計を受け取る API が未実装のコンパイル失敗を確認した。実装と既存呼び出し元の破棄 logger への移行後に handler 以下が Green となった。8種類の経路と、writer が失敗しても正常応答と枠の再利用を維持することを確認した。
3. 必須依存の欠落：ガードがない状態で logger・時計の nil を許してしまう Red を2状況で確認した。構築時のガード追加後に全通常テストが Green となった。
4. 既存の取得中・応答変換中の panic/cleanup テストへ、固定分類・発生段階・1回のログ・値の非漏出の検査を追加した。panic と同時枠・context の解放を維持した。
5. 実 SDK の4版で、初期化・一覧に完了ログを出さず、不正入力→通常または本文省略の収集→利用枠拒否の3回とログが対応することを確認した。実バイナリでも不正入力の安全なイベントが JSON/stderr へ出て、stdout は空で SIGTERM が正常終了することを確認した。実 Steam API は呼んでいない。

## 実行した検証と結果

Go は固定版 Docker Compose の verify、整形は format を使った。

- go test -count=1 ./internal/reviewmcp/adapter/handler/logging ./internal/reviewmcp/adapter/handler/protocol：成功。
- go test -count=1 ./internal/reviewmcp/adapter/handler/...：成功。
- go test -count=1 ./...：全通常テストとレイヤー・Controller 配置検査が成功した。
- go test -tags=integration -count=1 ./internal/reviewmcp/adapter ./cmd/server -run TestMCPCollectionLog：実 SDK 4版と実バイナリの stderr の確認が成功した。
- bash scripts/check-go.sh format-fix：整形を適用した。
- bash scripts/check-app.sh：最初は新しい結合テストの after-block の空行2箇所で停止した。修正後の再実行で整形・vet・通常テスト・golangci-lint（0 issues）、Git 非依存の3条件、結合を含む race、本番イメージの全項目が成功した。
- 同じ本番用イメージ sha256:b2d9ff51174b811a72269c8eabba8f96c52d4599fd5ab312beaa8e4b30357a4f で HTTP/MCP、Host/Origin、PORT、設定不正、SIGTERM/SIGINT を確認した。
- logging の通常テストだけを対象に、go test -count=1 -coverprofile と go tool cover -func を実行し、Write・status・causeKind と同パッケージ全体のステートメント網羅率100%を確認した。全アプリケーション・分岐・要求の網羅率ではない。最後の追加は結果未設定のテストのみであり、プロダクションコードは全検証の対象と同一である。
- 最初のコミットの pre-commit・commit-msg：秘密情報、規約ツール184テスト、Lefthook・actionlint・シェル構文、ステージ済み Go の整形とコミット形式が成功した。
- git diff --check：成功。source と差分のセルフレビューで、公開応答・開始回数・枠返却・親 context を変更しないことを確認した。

## 制約と未実施

OpenTelemetry の依存・trace・metrics・転送・保存基盤は追加していない。ログが永続保存されたことや、SDK の送信完了・実 ChatGPT・Render の動作は検証していない。任意の writer の非ブロッキング・保存成功は保証しない。

coverage 閾値・分岐測定・mutation・Stryker は未導入であり、Stryker を実行していない。実 Steam API、push・PR・CI は実行していない。ローカルの検証と配置環境の結果を区別する。
