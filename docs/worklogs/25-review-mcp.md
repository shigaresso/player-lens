# Issue #25：レビュー収集 MCP の作業記録

## 対象と方針

最新 origin/main `f516fc831f77b61289e01ba84105ac491692337e` から `codex/25-review-mcp` を作成した。[Issue #25](https://github.com/shigaresso/player-lens/issues/25) の数値上限と失敗契約を実装前に更新した。push・PR 作成・Issue クローズは行わない。

データ・調査仕様の収集条件と部分失敗、収集仕様の入力・走査・期限、HTTP サーバー仕様の期限と終了、公開デモ仕様の状態、開発手順、ADR-0001〜0006 を参照した。追加の仕様と採用理由は review-mcp.md と ADR-0007 に記録した。

## 依存と環境

公式 SDK v1.8.0 を固定し、Docker の Go 1.27.1 で `go get github.com/modelcontextprotocol/go-sdk/mcp@v1.8.0` を実行した。go.mod・go.sum、Docker の依存取得と Watch 対象を追加した。環境・仕様の変更には形式だけの Red を作らない。

## 実装・検証の状態

2026-10-09 JST に次の契約を確認した。TDD の関連コマンドは Compose の verify サービスで実行し、format サービスの固定版 gofmt で整形した。テスト追加→Red→本番コードのみの実装→Green の順を守り、対応テストと実装を同じローカルコミットにした。

| 契約 | 実際の Red | Green とコミット |
| --- | --- | --- |
| 入力・既定値・上限 | `go test -count=1 ./internal/reviewmcp` が parseInput 未定義で失敗した | 有効な条件・境界値・不正入力を確認した。46a942e |
| 収集結果の変換 | `-run TestCollect` が collect 未定義で失敗した | 既存の走査・重複除外・順序・時刻・空の配列と入力拒否を確認した。e41eba3 |
| 途中失敗・部分結果 | `-run TestCollectFailure` で部分結果が nil となり失敗した | fetch_failed・cancelled・deadline_exceeded、原因保持と内部情報を公開しないことを確認した。a980253 |
| 応答量 | `-run TestResponseSize` で140804バイトの結果が128 KiBを超えて失敗した | 小さい本文を保持し、超過は省略件数を明示した。0821224 |
| 実行期限 | `-run TestCollectionContext` で既定期限が2分となり失敗した | 8秒・短い親期限・値の伝播・開始前終了・解放を確認した。7733d2a |
| 実 SDK の接続 | `go test -tags=integration -count=1 ./internal/reviewmcp -run TestMCPProtocol` が NewHandler 未定義で失敗した | 4版の初期処理・一覧・呼び出し・入力エラー・未定義ツールを確認した。d7484eb |
| HTTP 切断 | `-run TestHTTPRequestCancellation` で4版とも切断が取得へ伝播せず失敗した | リクエストごとの context と SDK context の終了に従った。9b33ed9 |
| HTTP 本文・Origin | `-run 'TestHandlerLimits|TestNewHandler'` で413/403が400となり失敗した | 既知・未知の本文長と同一/外部 Origin を確認した。fad38a0。nil 依存の拒否は既に Green の回帰確認だった |
| サーバー経路 | `go test -count=1 ./cmd/server -run TestMCPRoute` が組み立て API の引数違いで失敗した | /mcp だけを委譲し healthz・終了の既存契約を維持した。ede0998 |
| 項目名の厳密さ | `-run TestParseInput` で大文字の未知項目を受け付けて失敗した | JSON decoder の大小文字による暗黙の受け入れを防いだ。976d345 |
| Docker 転送と Host | `-run TestLocalHost` で外部 Host とコンテナ IP を拒否せず失敗した | localhost とループバック IP のみを許可した。0eaf827 |

既存の言語検査は大文字化・前後空白除去をしないため、最初のテスト案を既存契約に合わせてから parseInput 未定義の Red を再確認した。SDK の往復では JSON のキー順が変わるため、結果の比較を JSON の値の同一性に修正した。仕様の期待結果は変えていない。

b5fb6d5 の追加検証は既に Green だったため、失敗を作らずテストだけのコミットにした。新旧クライアントへの部分結果・応答量超過・空の結果、本番イメージの MCP 初期化・一覧・不正入力、処理中の MCP 応答を待つサーバー終了を確認した。API キー・実 Steam・実 LLM は使っていない。

## 共通検査とセルフレビュー

- `bash scripts/check-app.sh`：整形・vet・通常テストは6パッケージで成功した。lint の ST1005（起動エラー文が Steam で始まる表記）で一度停止した。
- 指摘を修正した22611f3の後、`bash scripts/check-lint.sh`：0 issues。成功済みの検査全体を繰り返さず、残りの共通検査を個別に実行した。
- `bash scripts/check-go.sh integration`：Git の失敗・不在・checkout 不在の3状況、全6パッケージの `go test -tags=integration -race -count=1 ./...` が成功した。このコマンドは通常のテストも含む。
- `python3 scripts/check_image.py`：最終本番イメージ `sha256:2d1eef2f9c0515c021c28e36d9a0485ccdce2af13e7ba7420bfc1e63c5cfef76` を一度ビルドし、同じ ID で8080/10000の HTTP・MCP、外部 Host の拒否、SIGTERM/SIGINT、不正 PORT の非0終了を確認した。ソース共有なし・読み取り専用・非 root の既存検査も維持した。これはローカル arm64 の検証であり、未実行の CI・Render の検証を代用しない。
- `python3 scripts/verify_hooks.py`：実フックの8テストが成功した。165.355秒だった。pre-push の確認は一時リポジトリと差し替えた検査環境によるもので、作業ブランチを push していない。
- 各ローカルコミットの必須 pre-commit：`bash scripts/check.sh` の145テスト、差分・Lefthook・actionlint・シェル構文と index の Go 整形を確認した。commit-msg の規約検査も成功した。成功済みのフック検査を別の予備検査として重ねていない。

Docker の公開イメージ取得時は一時的な DOCKER_CONFIG と Docker context の socket を子プロセスへ渡した。ユーザーの Docker 設定・認証や Git の safe.directory を変更していない。レビュー・トークン・設定ファイル・実行ファイルは Git に追加していない。

AI によるセルフレビューで、I/O の組み立て・元の収集操作の再利用・純粋な JSON 変換、失敗の cause とサイズ、context・接続プール・goroutine の所有、Docker と CI の同じ検査経路を確認した。人によるコードレビュー、push 後の実 CI、coverage・分岐・mutation は未実施/未導入と区別する。

## ChatGPT の手動確認

ChatGPT アプリへのサーバー登録と再起動はユーザーの操作が必要であり、アプリ設定を自動変更していない。したがって、ChatGPT での実接続・実 Steam の取得・モデルの分析回答は未確認である。[MCP デバッグ手順](../mcp-debug.md) に localhost の登録、表示ポート、App ID 440・日本語3件・1ページの確認例と、引用 ID の照合を記載した。SDK と本番イメージでの自動検証の成功を、ChatGPT での分析確認済みと扱わない。

手動確認後は日時・取得条件・件数・終了理由・引用 ID の対応を追記する。秘密情報・全文レビューは記録しない。レビューの保存・バックエンドの LLM 実行・公開配置は導入していない。


## 追加の実 Steam 確認

2026-10-09T02:26:57+09:00 の「実際に実行してみたい」というユーザー指示で、`python3 scripts/dev.py` を起動した。空きポート8080で healthcheck が成功し、Compose Watch を継続した。接続 URL は `http://127.0.0.1:8080/mcp`。

Python 標準の HTTP クライアントで MCP 2025-11-25 の initialize・tools/list・tools/call を実行した。取得先を差し替えず、本番の Steam Client を通して App ID 440・language=japanese・max_reviews=3・max_pages=1 のレビューを取得した。

- isError=false、error_code は空、returned_count=3、pages_fetched=1、stop_reason=review_limit。
- 呼び出しは約0.49秒で成功した。レビュー ID は236407859・235397710・235116218だった。本文を検証記録やファイルへ保存していない。
- Steam の通信は1ページの1リクエストに限定した。通常テスト・CI に実 Steam 通信を追加していない。
- ChatGPT アプリへの登録と、アプリからのツール呼び出し・分析回答は引き続き未確認。上記はローカル HTTP クライアントからの実 API 確認である。

サーバーはユーザーが接続を試すため起動を継続した。push・PR 作成・Issue クローズは行っていない。
