# MCP 入力の解析と条件生成を同じファイルにまとめた

[Issue #115](https://github.com/shigaresso/player-lens/issues/115) に対応した。PR #114 統合済みの origin/main（d9b018b）から codex/115-consolidate-mcp-input を作成した。

input_options.go の定数と normalizeToolInput を input.go へ移し、定数→ParseInput→normalizeToolInput の順に配置した。既存の TestNormalizeToolInput も input_test.go に移し、旧実装・テストのファイルを削除した。独自 JSON 読み取りの tool_input.go、各関数の責務とシグニチャ・本文・定数値、入力とエラーの契約は維持した。

関数の分離とファイルの分離を区別し、今回の短い入力処理と関連テストを一緒に読める配置へ整理した。テストの1対1対応や新しい配置規則を一律に導入していない。VS Code の既存の汎用ネスティング設定で input_test.go は input.go の下に対応するため、設定は変更していない。MCP 仕様の入力変換の節に実装の参照と配置理由を反映した。新しい ADR は追加していない。

## 検証

動作を変えない移動のため形式的な Red や新規テストは追加していない。変更前に固定版 Compose verify の go test -count=1 ./internal/reviewmcp/adapter/handler/protocol -run '^(TestParseInput|TestNormalizeToolInput)$' が成功した。

移動後に origin/main のソースと比較し、ParseInput・normalizeToolInput の本文、4つの定数、TestParseInput・TestNormalizeToolInput の関数全体が同一であることを確認した。MCP の既定値・上限、失敗時のゼロ値・原因と文言、呼び出し元の入力の非変更を変更していない。

bash scripts/check-app.sh は終了コード0で成功した。整形差分・go vet ./...・go test -count=1 ./...、通常と integration タグの lint（0 issues）、Git 非依存3条件、go test -tags=integration -race -count=1 ./... を確認した。最後のコマンドは通常と結合の両方を含む。

本番用 image ID sha256:4bf6d1b84fca97e4f0eabb6cf4549434033bc2987cb62ef18bfe0e93316cfd65 を一度ビルドした。同じイメージで非 root・ソース共有なしの起動、HTTP/MCP、ローカル／公開 Host・Origin、PORT、不正設定の非0終了、SIGTERM／SIGINT の正常終了を確認した。

## 参照と確認範囲

[MCP 仕様](../specs/review-mcp.md) の「入力変換の境界」、文書一覧、データ・調査仕様の導入状況、操作一覧の入力変換、ADR-0013 の責務と配置、ADR-0015 の生成済み収集条件を参照した。開発手順と ADR-0002・0003 の既存の検証経路を維持した。

GitHub 認証・Docker・Git の書き込みは許可された権限で実行した。新しい通信・保存先・生成データは追加せず、検証ログはリポジトリ外の一時ディレクトリに置いた。必須フックの結果と確定したコミットは Issue に記録する。

AI によるセルフレビューと文書リンク・端末パス非混入の確認を行い、人によるレビューは未実施とした。push・PR 作成・Issue のクローズ、GitHub CI・実 Steam／LLM・Render は未実施。coverage 閾値・分岐網羅・mutation（Stryker を含む）は現行では未導入であり、実行済みの検査とは区別する。
