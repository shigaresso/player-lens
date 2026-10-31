# Steam クエリ生成関数を new の命名に揃えた

[Issue #127](https://github.com/shigaresso/player-lens/issues/127) に対応した。PR #126 統合済みの origin/main（3eeea21）から codex/127-query-constructor-name を作成した。

buildReviewPageQuery を newReviewPageQuery、TestBuildReviewPageQuery を TestNewReviewPageQuery に改名し、呼び出し元と説明コメント、現行の1ページ取得仕様・ADR-0004・操作一覧を更新した。値の生成を newReviewPageAPIInput・newReviewPageHTTPRequest と同じ new の名前で表すようにした。過去の作業記録は当時の名前を保持した。

API 入力の JSON 化と url.Values の生成を担当し、URL エンコードと HTTP Request の生成は newReviewPageHTTPRequest、送信は FetchReviewsPage に維持した。入出力・JSON キーと値・エラー文言・入力検査・期限・キャンセル・Body の解放・公開 API は変更していない。

## 検証

振る舞いを変えない改名のため、形式的な Red や新しいテストは追加していない。変更した Go コードと既存テストの2ファイルを origin/main と照合し、識別子の置換以外が完全に同一であることを確認した。入力・期待値・状況名・アサーションを維持し、現行の Go コード・仕様・ADR・操作一覧に旧識別子が残っていないことを確認した。

bash scripts/check-app.sh が終了コード0で成功した。固定版 Compose の整形検査、go vet ./...、go test -count=1 ./...、通常・integration タグの lint（0 issues）、Git 非依存3条件、go test -tags=integration -race -count=1 ./... を確認した。最後のコマンドは通常・結合の両方を含む。

本番用 image ID sha256:4110a44c90640b8e89094e3bfabc0e6014f0f09188785b11b1974afb1a745b62 を一度ビルドし、同じイメージで HTTP/MCP、ローカル／公開の Host・Origin、PORT、SIGTERM／SIGINT、不正な公開 URL・PORT・利用制限設定の非0終了を確認した。ソース共有なし・非 root の既存検証経路を使用した。

git diff --check、差分のセルフレビュー、文書リンクと端末固有情報の非混入を確認した。必須コミットフックの結果と確定コミット・コミット範囲の規約検査は Issue に記録する。

## 参照と確認範囲

文書一覧、1ページ取得仕様の「リクエスト変換の境界」「HTTP リクエストの組み立て」「検証」、データ・調査仕様の収集と導入状況、ADR-0004・ADR-0012 の境界、操作一覧、開発手順と既存の検証経路を参照した。

GitHub 認証・Docker・Git の書き込みは許可された権限で実行し、ログはリポジトリ外の一時ディレクトリへ置いた。新しい保存先・実データ・依存は追加していない。

push・PR 作成・Issue のクローズ、GitHub CI・実 Steam／LLM・Render の検証、人によるレビューは未実施。coverage 閾値・分岐網羅・mutation（Stryker を含む）は現行では未導入であり、今回の確認結果に含めない。
