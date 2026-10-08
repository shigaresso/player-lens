# Issue #27：MCP 収集の利用制限

## 範囲と参照

[Issue #27](https://github.com/shigaresso/player-lens/issues/27) に対応した。PR #26 の統合後に最新 origin/main `8050ae872bcccdb6bb8408238e157852f5d41b02` から `codex/27-mcp-usage-limits` を作った。push 先の upstream は設定していない。

文書一覧に従い、MCP 仕様の接続・入力・結果・期限と資源、データ・調査仕様の上限と部分失敗、公開デモ仕様の制御と README、HTTP サーバー仕様の設定・終了、ユースケースの入口の境界を参照した。開発手順と採用済み ADR-0001・0002・0003・0007 の既読の方針を継続して適用し、利用制限の具体的な契約を [仕様](../specs/mcp-usage-limits.md) と [ADR-0008](../adr/0008-mcp-usage-limits.md) に記載した。

## 実装と契約単位の確認

| 契約 | 実際の Red | Green とコミット |
| --- | --- | --- |
| 初期値・環境変数・不正設定の起動拒否 | Compose verify の `go test -count=1 ./cmd/server ./internal/reviewmcp -run 'TestLimits\|TestReadMCPLimits'` が Limits・DefaultLimits・readMCPLimits 未定義で失敗した | 両パッケージの通常テストが成功した。既定値、正規化、各設定の0・負数・符号・空白・小数・文字列・範囲外を確認した。24c3a16 |
| 分次・24時間の固定窓と回数の消費 | `go test -count=1 ./internal/reviewmcp -run 'TestLimiterWindows\|TestCollectionUsageBudget'` が newLimiter 未定義と collect の引数違いで失敗した | 関連パッケージの通常テストが成功した。時計差し替えで直前・ちょうど・直後、独立した窓、初回受付の起点、拒否・不正入力・開始前キャンセルの非消費を確認した。9e7a03a |
| 同時実行の受付と枠返却 | `go test -count=1 ./internal/reviewmcp -run 'TestLimiterConcurrentAdmission\|TestLimiterBusyDoesNotConsumeBudget\|TestCollectionReleasesSlotAfterFailure'` が acquire 未実装で失敗した | `go test -race -count=1 ./internal/reviewmcp` が成功した。80件の並行受付で3件だけ許可し、busy・二重返却・途中失敗・実行中キャンセル・期限・応答量超過の後始末と開始回数の非返却を確認した。6339b9d |

上記のコマンドは固定版 Go の Compose verify 内で実行した。テストだけを追加して未実装の API による Red を確認し、実装とテストを契約ごとに一緒にコミットした。既存テストの呼び出し引数の更新は機械的な変更として同じコミットに含めた。入力違いごとのコミットや Red 状態の共有は行っていない。

実 SDK の追加確認はすでに Green だったため、失敗を作らず検証のみを追加した。`go test -tags=integration -race -count=1 ./internal/reviewmcp -run 'TestMCPShared'` で4版の別セッション・異なる版による枠の共有、初期化・一覧・不正入力・未定義ツールの非消費、実行中の拒否・完了後の再受付を確認した。セルフレビューでクライアント goroutine の完了待機をエラー経路にも明示し、変更した関連テストを再実行して成功した。本番イメージの検査にも3つの不正な MCP 環境変数の非0終了を加え、6b5857e にまとめた。

## 共通検査とセルフレビュー

`bash scripts/check-app.sh` が成功した。

- 固定版 Go の整形差分・vet・全6パッケージの通常テストが成功した。
- 固定版 golangci-lint は0 issuesだった。
- Git 失敗・Git 不在・checkout 不在の3状況と、全6パッケージの `go test -tags=integration -race -count=1 ./...` が成功した。通常テストも含むコマンドである。
- 本番 image ID `sha256:b758f6bc045f9a6afbabca968e4a919e149dfdf8dc923dc5596ebff434109793` を一度ビルドし、同じ ID で8080/10000の HTTP・MCP、SIGTERM/SIGINT の正常終了、不正 PORT と3つの不正 MCP 設定の非0終了を確認した。ソース共有なし・読み取り専用・非 root の検査を維持した。
- 変更した結合テストの後始末は上記の通り関連テストを再確認した。プロダクションコードは本番イメージ検証後に変更していない。

各コミットの必須 pre-commit と commit-msg を実行した。`bash scripts/check.sh` の145テスト、差分・Lefthook・actionlint・シェル構文と index の Go 整形、コミット形式の検査が成功した。フック自体は変更していないため、Issue #25 で成功した `python3 scripts/verify_hooks.py` の8テストを、予備検査として再度実行していない。作業ブランチの push による pre-push と実 CI は未実施とする。

AI によるセルフレビューで、同じ handler の状態共有、mutex の判定・更新、defer と二重返却防止、context・goroutine の後始末、制限拒否時の取得0回、CLI・healthz・Host・Origin・既存の期限・応答量の維持、文書と設定の対応を確認した。人によるコードレビュー、公開配置、coverage・分岐・mutation は未実施/未導入である。レビュー本文・秘密情報・ビルド成果物を Git に追加していない。

Docker の検査は一時的な DOCKER_CONFIG と既存 Docker context の socket を子プロセスへ渡して実行した。ユーザーの Docker 認証設定や Git 設定の共有値を変更していない。

## 登録済み MCP ツールの実接続

2026-10-09 JST に、ユーザーが登録・再起動した MCP の `collect_reviews` を、この会話の登録済みツール経由で呼び出した。App ID 4800590、language=all、max_reviews=3、max_pages=1 を指定し、取得先は本物の Steam だった。

- returned_count=3、pages_fetched=1、stop_reason=review_limit、error_code は空だった。
- レビュー ID は237211330・237210202・237181053で、3件とも推奨だった。
- モデルが本文を読み、スタイルへの好感・気軽に遊べる点・楽しさへの評価を ID と対応させて回答した。3件の少量取得であり、全体の評価や代表性を確認したものではない。
- これは登録済み MCP ツールの呼び出しである。以前の Python HTTP クライアントによる確認や、自動テストの SDK と区別する。アプリ設定を自動変更していない。レビュー全文をファイル・DB・検証記録に保存していない。
- この確認は利用制限の追加前に行った。利用制限の追加後は取得を差し替えた自動検証を行い、実 Steam やモデルの追加呼び出しを繰り返していない。

## 残る制約と停止位置

利用制限は1つの handler / プロセスのメモリ内で全利用者が共有する。再起動で回数が消え、複数台では台数分の枠がある。固定窓の境界付近の集中、拒否応答やインフラの料金、クラウドの請求額の上限は保証しない。

公開 Host、認証・利用期限・利用者ごとの割り当て、永続した共有予算、Render/AWS への配置、バックエンドの LLM 実行は別 Issue とする。仕様と ADR にこの範囲を記録した。ローカルコミットで止め、push・PR 作成・Issue クローズは行わない。
