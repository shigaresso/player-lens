# MCP の応答変換完了まで同時実行枠を保持するよう修正した

## 背景・目的

[Issue #62](https://github.com/shigaresso/player-lens/issues/62) に対応した。Issue #56 のレイヤー分類で、収集と枠管理を application へ移した際、defer release が adapter の応答変換より先に実行されるようになった。ADR-0008 と利用制限仕様の「応答変換完了まで保持する」契約へ戻した。実運用の障害報告ではなく、コードと履歴から検出した不整合である。

最新 origin/main（3d864dc54d259427cb8c64b3a82aab10247db82e、PR #61 統合済み）から codex/62-mcp-response-slot-lifetime を作成した。既存ブランチは維持し、upstream は設定していない。push・PR 作成・Issue のクローズは行わず、ローカルで確認できる状態にする。

## 変更内容

- application の CollectReviews に同期的な結果処理関数 func(Result) error を渡し、枠の確保・収集・応答生成・defer による返却を同じ実行範囲にした。成功・取得失敗・受付拒否・開始前キャンセルの結果を一度だけ渡す。
- adapter は結果処理関数で既存の失敗分類・JSON・応答量判定・SDK 応答生成を実行する。application は SDK と adapter に依存せず、返却関数を呼び出し側へ渡していない。既存の Result とエラー変換・部分結果を維持した。
- 結果処理のエラーはそのまま返し、収集の原因は Result.Err に保持して既存の変換へ渡す。変換失敗でも確保済みの枠は返す。開始回数は戻さず、未確保の枠を返さない。取得用 context は操作が解放する。
- 利用制限仕様、MCP の結果変換、操作一覧と ADR-0013 の記述を整合させ、ADR-0016 に同期的な結果処理を選ぶ理由を記録した。HTTP 送信完了までは保持範囲へ追加せず、JSON 変換を8秒で強制中断する保証も追加していない。

## 検証

### Red・Green とコミット

変更前に Docker Compose の verify で `go test -count=1 ./internal/reviewmcp/...` を実行し、3パッケージの既存テストが成功した。

テストだけを追加し、`go test -count=1 ./internal/reviewmcp/adapter -run '^TestCollectKeepsSlotDuringFailureConversion$'` を同じ verify で実行した。実際の adapter が取得失敗を公開情報へ分類する時点で、別の確保が成功してしまい「応答変換中に同時枠が空いている: reason=0」で Red になった。API 未定義による失敗ではなく、変更前の動作の誤りを直接検出した。

その後に application と adapter の実装を変更し、必要な既存テストの呼び出しを新しい API へ移行した。移行で不要になった import を除いた後、同じ MCP 全体のテストが Green になった。修正・直接の回帰テスト・必要な参照移行・対応仕様を aa023dd にまとめた。

追加の application 契約テストと実 SDK の回帰テストは、実装を変更せずに追加した。最初の application テストでは、Steam 収集が所有する子 context も応答変換まで有効だと誤って期待していた。子 context は Steam 収集の終了時に解放する既存の契約であり、その誤ったアサーションを除いた。終了後の解放と親 context の非干渉は維持した。期待を追加するためにプロダクションコードは変更していない。

`go test -tags=integration -race -count=1 ./internal/reviewmcp/application ./internal/reviewmcp/adapter -run '^(TestCollectReviewsResponseLifetime|TestCollectReviewsResponseFailureWithoutAcquisition|TestMCPSlotDuringResponseConversion)$'` が成功した。追加契約は修正時点で実装済みなので、形式上の Red は作らず、テストのみの 1a1b5e8 に分けた。

### 追加の panic 検証

ユーザーの追加依頼により、TestCollectReviewsPanicReleasesSlot を追加した。同じ goroutine の収集中・応答変換中に意図した panic を起こす2状況をテーブルで確認し、テスト側で recover して原因の同一性を確認する。通常のエラーとして返したり panic を握りつぶしたりせず、枠を再確保でき、開始回数は戻らず、取得用 context が解放されて親はキャンセルされないことを確認した。

`go test -race -count=1 ./internal/reviewmcp/application` を Docker Compose の verify で実行し、追加時に Green だった。プロダクションコードは変更せず、形式上の Red は作っていない。追加後の `bash scripts/check-go.sh format`、`bash scripts/check-lint.sh` と必須フックを確認した。既存の結合・本番用イメージの検証結果は上記の実行結果を維持し、今回再実行したとは扱わない。

### 契約とテストの対応

| 対象 | 確認した契約 |
| --- | --- |
| TestCollectKeepsSlotDuringFailureConversion | 実際の失敗応答の分類中に枠を保持し、変換後に返す。原因の Is を観測するテスト用エラーを使い、プロダクションコードにテスト用の差し込み口を増やしていない |
| TestCollectReviewsResponseLifetime（4状況） | 収集成功／失敗と結果処理成功／失敗の組み合わせで、処理中の枠保持、一度だけの同期呼び出し、収集原因と結果処理エラーの区別、終了後の枠・context 解放、開始回数を戻さないこと |
| TestCollectReviewsResponseFailureWithoutAcquisition（2状況） | 受付拒否と開始前キャンセルの結果処理が失敗しても、他の処理の枠を返さず、未確保の枠・開始回数を消費しないこと |
| TestCollectReviewsPanicReleasesSlot（2状況） | 収集中・応答変換中の panic の原因伝播、枠と context の解放、開始回数を戻さないこと |
| TestMCPSlotDuringResponseConversion（4版） | 実 SDK の応答変換を制御した時点で別セッションの呼び出しが busy になり、変換完了後に再受付できること。全呼び出しはローカルサーバーと取得のテスト実装を使う |

結合テストはチャネルで変換開始・続行・クライアント処理完了を同期し、時間経過で変換時点を推測していない。失敗時も続行・キャンセル・goroutine の完了待ちとセッション・サーバーの終了を行う。

### 全体の検証

`bash scripts/check-app.sh` が終了コード0で成功した。固定版 Docker Compose で整形・vet・通常テスト、integration タグを含む lint（0 issues）、Git 非依存の3条件、race 付き通常／結合、本番用イメージを確認した。通常は全15パッケージを収集して14で実行し、cmd/collect-reviews の root は integration タグのプロセステストで検証した。race 付き通常／結合は全15パッケージで成功した。

本番用 image ID `sha256:1260eaa8ab54e0e415944c94f5de74f84f8f95db74fc937b9d51da84488ef7b8` を一度ビルドし、同じイメージでローカル／公開設定の HTTP/MCP、Host・Origin の拒否、PORT、非 root・読み取り専用・ソース共有なしの起動、不正設定の非0終了、SIGTERM/SIGINT の正常終了を確認した。全体検証後のプロダクション変更は Go の契約コメントの明確化だけで、振る舞いは変更していない。

必須 pre-commit の Python177テスト・静的検査・index 整形と commit-msg を各コミットで実行した。最終確認にはコミット範囲、差分、PR 本文として使えるこの記録と Issue 本文のローカル規約検査を使う。

### AI のセルフレビューと制約

defer の範囲に実際の応答変換を含め、早期返却と返却漏れの両方を確認した。確保に成功した経路だけに defer があり、拒否と開始前キャンセルに返却を追加していない。Result.Err と結果処理の error を区別し、公開 JSON と内部原因の非公開契約を既存の変換・結合テストで維持した。

既存テストの期待値と入力の値は変更せず、呼び出し API と結果捕捉のヘルパーを移行した。既存のテスト・品質条件を除外して通していない。レイヤー依存の検査が成功し、新しい依存や SDK の application への import はない。人によるレビューは未実施。

GitHub 認証と Docker は権限付きで実行した。前回の一時 Docker 設定を再利用し、`DOCKER_CONFIG=/private/tmp/player-lens-58-docker-config DOCKER_HOST=unix:///Users/soregashi/.docker/run/docker.sock` を Docker とコミットフックへ渡した。既存の Docker 認証設定とフックを変更・省略していない。

push 前のためコード CI と、PR 作成 CLI の送信済み head/base による検査は未実施で、CI の完了待機もしない。実 Steam/LLM/ChatGPT・配置も実施していない。coverage 閾値・分岐・mutation（Stryker を含む）は未導入であり、今回の検査成功をそれらの品質目標達成とは扱わない。

## 参照仕様・ADR

- docs/specs/mcp-usage-limits.md「受付と固定窓」「検証と保証の範囲」、ADR-0008 の結果変換完了後の返却。
- docs/specs/review-mcp.md「結果・失敗・資源」「結果変換の境界」、ADR-0007・0013・0016 の application と adapter の境界。
- docs/specs/collect-reviews.md の期限・部分失敗、data-and-research.md の収集・上限、ADR-0005・0014・0015。
- docs/use-cases.md、docs/development.md、ADR-0001〜0003 の既存の検証・フック環境。

## 関連 Issue

Closes #62
