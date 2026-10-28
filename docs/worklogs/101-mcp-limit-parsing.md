# MCP 利用上限の読み取りと解析を分離した

## 背景・目的

[Issue #101](https://github.com/shigaresso/player-lens/issues/101) に対応した。PR #100 統合済みの最新 origin/main（8e3ed1e）から codex/101-separate-mcp-limit-parsing を作成した。既存のブランチと作業履歴を保持し、upstream は設定していない。

## 変更内容

同じ cmd/server/adapter/limits.go の中で、公開する ReadMCPLimits、取得済みの文字列を保持する非公開の mcpLimitsInput、純粋な parseMCPLimits に分けた。公開関数から、3項目の読み取り・解析・NewLimits による生成を順に読めるようにした。

解析関数は既定値を適用し、符号・空白のない31 bit の10進数へ変換して LimitsConfig を返す。環境変数の取得関数や I/O の依存は持たず、正数条件は既存の NewLimits に維持した。環境変数名は同じファイルの非公開定数を読み取りと診断で共有する。別ファイルへの実装分割、新しいパッケージ・interface・公開 API は追加していない。

3項目を分次・24時間・同時数の順に各1回取得し終えてから解析する構成に変わるため、以前の途中解析失敗時の読み取り打ち切りとは異なる。実際の取得元は既存の os.Getenv を使い、既定値・数値の範囲・先頭のゼロ・エラーの説明と原因保持、すべての数値変換を行ってから正数を検査する優先順は維持した。利用枠・時計・HTTP・終了・資源の解放は変更していない。

## 検証

### 既存の契約の Green

本番コードを変更する前に、TestReadMCPLimits の既存テーブルへ個別指定と上下限を追加し、読み取る環境変数名・順序・回数、変換原因とドメインの正数条件、複数の不正値の優先順を確認するテストを追加した。

Docker Compose verify で go test -count=1 ./cmd/server/adapter -run '^TestReadMCPLimits' の Green を確認した。既存の契約を補強する回帰テストなので形式的な Red は作らず、テストだけを 642f46e にコミットした。pre-commit の秘密情報・Python 182テスト・静的検査・index の Go 整形と、commit-msg の秘密情報・規約検査が成功した。

### 解析の Red と Green

TestParseMCPLimits を追加し、mcpLimitsInput と parseMCPLimits が未定義であるコンパイル失敗の Red を確認した。取得済み文字列からの既定値・対応する整数への変換・先頭のゼロ・正数条件を生成側へ委ねる0、構文不正・後続項目の範囲超過でゼロ値の設定と原因を返す契約を対象にした。

本番コードだけを追加し、同じテストと既存の ReadMCPLimits のテストが Green になることを確認した。

### 読み取りと解析の接続の Red と Green

TestReadMCPLimitsReadsBeforeParsing を追加し、最初の値が不正だと読み取りが1項目で終わるため失敗する Red を確認した。その後、公開関数で3項目の文字列を取得してから解析と生成を呼ぶように本番コードだけを変更した。

最終の関連テストは、Docker Compose verify で go test -count=1 ./cmd/server/adapter -run '^(TestParseMCPLimits|TestReadMCPLimits.*)$' の Green を確認した。各 Red・Green で -count=1 を指定した。

### 全体検証

bash scripts/check-app.sh が終了コード0で成功した。

- gofmt の差分検査、go vet ./...、go test -count=1 ./... が成功した。
- 固定版 golangci-lint は通常と integration タグのコードを検査し、0 issuesだった。
- Git の失敗・不在・checkout の不在という3条件で、Go の実行と子ビルドが成功した。
- go test -tags=integration -race -count=1 ./... が成功した。通常テストと結合テストの両方を含む。既存の起動失敗・待ち受け・HTTP・接続解放・終了と、ローカルの実 MCP SDK 連携を確認した。
- 本番用 image ID sha256:91ef6796cc62a12379761e6d7bc759e51240034db9b56aa010e23608c9009e16 を一度ビルドし、同じイメージの非 root 設定、ソース共有なしの起動、HTTP/MCP・ローカル／公開 Host・Origin、PORT、不正な3つの MCP 設定それぞれの非0終了、SIGTERM／SIGINT の正常終了を確認した。実 Steam は呼んでいない。
- 変更文書のローカルリンク13件の参照先が存在し、変更内容に端末のホームパスが含まれないことを確認した。リンクのアンカーは対象外とした。

### AI のセルフレビューと未実施項目

環境変数の名前は非公開定数の1定義にまとめ、読み取り側と診断の対応を揃えた。読み取りと解析を同じループに残さず、解析関数は入力の文字列だけを扱う。0の数値変換と正数条件の拒否、構文不正・範囲超過の原因、分次・24時間・同時数の優先順を分離後もテストで確認した。取得済みの入力は書き換えず、解析失敗で部分的な設定を返さない。人によるレビューは未実施。

Go の実行と必須フックの Docker は、前回の作業で用意した認証情報のない一時 DOCKER_CONFIG と既存の Docker ソケットを使った。GitHub 認証・Docker・Git の書き込みは権限付きで実行し、フック・対象・設定を省略・緩和していない。一時ログ・PR 用本文はリポジトリ外に置き、Git・Docker のビルドコンテキストに含めていない。

push・PR 作成・Issue のクローズは行っていない。pre-push・送信済み head/base の PR 作成検査・コード CI は未実施とする。実 Steam／LLM 接続・Render 配置は対象外とした。coverage 閾値・分岐・mutation（Stryker を含む）は現行で未導入であり、今回の結果を品質目標の100%達成とは扱わない。

## 参照仕様・ADR

- [MCP 利用制限仕様](../specs/mcp-usage-limits.md) の「設定」「設定値の生成と検証」「検証と保証の範囲」
- [ADR-0008](../adr/0008-mcp-usage-limits.md) の入力境界と正数条件、[ADR-0013](../adr/0013-domain-folder-layers.md) の設定解析の配置
- [開発手順](../development.md) の「Go・Compose・本番用イメージの検査」「各フックの責務」、ADR-0001・0002・0003 の既存の実行・検証方針

## 関連 Issue

Closes #101
