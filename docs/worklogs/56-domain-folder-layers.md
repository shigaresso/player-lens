# 各領域フォルダの内部をレイヤーで分類した

## 背景・目的

[Issue #56](https://github.com/shigaresso/player-lens/issues/56) に対応した。ユーザーの意図に従い、既存の各領域フォルダを維持し、その内部を domain・application・adapter で分類した。

以前の internal/review へ統合した3コミットはユーザーの指示で取り消した。分類軸もユーザーの回答で domain・application・adapter と確認し直し、Issue の方針と完了条件を更新した。今回の履歴は最新 origin/main（7bcc5cf578bff97cd1cefee770e2f3048453908a）からの変更だけを含む。ブランチは `codex/56-review-package-layout` を使用し、push・PR 作成・Issue のクローズは行わない。

## 変更内容

- steam は domain に型・取得条件・結果と正規化、application に収集・収集保存と利用側 interface、adapter に HTTP と要求・応答変換を配置した。ReviewType の基底型・公式定数に基づく定義・数値は維持した。
- reviewmcp は domain に上限と受付・返却、application に利用枠・最大8秒の期限と収集、adapter に HTTP/SDK・入力・公開 schema・結果・ガイドを配置した。application の Result は収集結果・拒否理由・元の失敗を分け、SDK の型を持たない。adapter は従来と同じ JSON・GetError の原因保持へ変換した。
- reviewcli と jsonoutput は各 adapter に移した。業務操作は steam/application を使用し、空のレイヤーや委譲だけのサービスは作っていない。
- mcpaccess は domain に URL と Host の設定・照合、adapter に HTTP 保護を配置した。Access の IsInitialized・AllowsHost は内部表現を隠したまま利用でき、Protect は設定と次の handler を受け取る関数へ移した。nil・ゼロ値・Host・Origin の既存の拒否契約を維持した。
- 各 cmd は main と起動の組み立てを維持し、I/O と表示をそれぞれの adapter に置いた。signal・context・listener・接続・goroutine の所有と終了担当は維持した。
- 既存のテストとガイドを移動先へ置き、イメージ検査とその fixture・仕様・ユースケース・README・文書一覧を更新した。ADR-0013 に各領域内のレイヤー配置と依存を記録し、以前の配置判断の変更点を明示した。

## 検証

### 移動と application の契約

動作を維持する移動・改名には形式上の Red を作っていない。移動後の通常テストが成功した。移動時の未使用 import・外部テストの参照漏れはコンパイル・整形検査で確認して修正した。

MCP application の TestCollectReviews を先に追加し、`docker compose run --rm --no-deps -T verify go test -count=1 ./internal/reviewmcp/application` が未実装の application（non-test Go ファイルなし）で失敗する Red を確認した。利用枠・8秒の期限・収集の実行を抽出して成功を確認した。新規テストの初期設定では分次・日次が同時に到達しており、既存どおり日次が優先されたため、分次の非返却を検査する設定を日次10回へ修正した。既存テストの期待は変更していない。

全体の `go test -count=1 ./...` と `bash scripts/check-app.sh` が成功した。後者は整形・vet・通常テスト、integration タグを含む lint（0 issues）、Git 非依存3条件、race 付き通常／結合テストを含む。本番用 image ID `sha256:12201da1e3dfb827e3706d903f7738156441e9fc084b82fbd3eef8876e64038b` を一度ビルドし、同じイメージで HTTP/MCP、ローカル／公開設定、Host・Origin、PORT、不正設定、SIGTERM/SIGINT の終了を確認した。これらの本番コードとイメージ検査は、その後のテスト追加で変更していない。

レイヤー分離と初回の契約テストを 596eeab にまとめた。必須 pre-commit の規約ツール177テスト・静的検査・index の整形と commit-msg が成功した。

### 追加した公開契約の回帰テスト

MCP application の分次・日次・同時枠の3種類の拒否、取得途中の部分結果と原因保持、開始前キャンセルの非消費を直接確認するテストを追加した。`docker compose run --rm --no-deps -T verify go test -race -count=1 ./internal/reviewmcp/application` が既存の実装で Green となり、追加の実装変更や形式上の Red は作っていない。

Access の domain には nil・ゼロ値・初期化済み、公開 Host の正規化・ローカル Host・未設定と不正な Host の6状況を追加した。HTTP と SDK を使わず、公開した設定・照合の契約を確認した。

### 依存境界の検査

internal/architecture_test.go は cmd と internal の Go ファイルを自動収集する。domain と application から具体的な adapter/SDK を参照しないこと、分類前の平坦なパッケージへの import を残さないことを検査する。MCP から Steam の型・利用側契約への依存だけを決定した境界として許可した。

本番コードに testing を import するパッケージをテスト helper として検出し、他の本番コードからの参照を拒否する。実行するテストを手動の許可リストで絞っていない。

`docker compose run --rm --no-deps -T verify go test -count=1 ./internal` は追加時から Green だった。検査自体の確認として Steam domain→SDK、Steam application→adapter、MCP application→adapter、アクセス domain→adapter の4違反と、本番 cmd→テスト helper の1違反を一時ファイルで与えた。各ファイルと理由を示す失敗を確認し、finally で全ファイルと作成した一時ディレクトリを除去した。同じコマンドで Green を確認し、違反ファイル・検査除外は残していない。

### 追加後の全体検証と履歴

全テスト追加後に `bash scripts/check-go.sh unit`、`bash scripts/check-lint.sh`、`bash scripts/check-go.sh integration` が成功した。通常は全15パッケージを収集して14でテストを実行した。cmd/collect-reviews の root は通常テストを持たず、integration タグ付きのコマンド実行テストで検証した。race 付き通常／結合は全15パッケージで成功した。lint は integration タグ付きテストを含めて0 issues、Git 非依存3条件も成功した。

公開契約の追加テストを fc64cc5 に分け、必須 pre-commit（規約ツール177テスト・静的検査・index 整形）と commit-msg が成功した。依存境界の検査と最終記録は独立した変更単位にまとめる。本番コードとイメージ検査に追加変更がないため、成功した本番用イメージ検証を繰り返していない。

### AI のセルフレビュー

既存のテスト関数118件と、各 Test 関数内の文字列・数値リテラルが移行前と一致することを読み取りスクリプトで確認した。公開 JSON・schema・分析ガイド・上限・既定値、完全一致する入力キーの扱いを維持した。テストのリテラル比較は補助確認であり、実際の動作は実行したテストで確認した。

application に HTTP/SDK への依存がないこと、MCP の入力拒否と受付の順序、期限と context の解放、利用枠と開始回数の返却の違い、失敗・拒否時の GetError と JSON の扱いを読み、抽出前と同じであることを確認した。Host の正規化・localhost とループバック・公開 URL の拒否、サーバーと CLI の起動順序・終了コード・I/O の解放担当も確認した。人によるレビューは未実施。

### 未実施と保証範囲

コード CI は push 前のため未実施で、CI の完了待機も行っていない。PR 作成 CLI の送信済み head/base を用いる検査、実 Steam・LLM・ChatGPT 接続・配置も今回実施していない。coverage 閾値・分岐・mutation（Stryker を含む）は未導入であり、既存検査の成功をそれらの品質目標達成とは扱わない。

## 参照仕様・ADR

- docs/specs/collect-reviews.md「範囲と依存」「入力と正規化」「ページの走査と結果」「期限と失敗」、steam-review-page.md「API と入力」「結果と失敗」「Steam の名前付き数値型」、review-cli.md「CLI の入力と終了」「保存の順序と失敗」「JSON 変換と writer」、ADR-0004〜0006・0012。
- docs/specs/review-mcp.md「接続と入力」「結果・失敗・資源」「呼び出し回数と同時数」、mcp-usage-limits.md「設定」「受付と固定窓」「拒否応答」、mcp-public-access.md の URL・Host 保護、review-analysis.md のガイド提供、ADR-0007〜0009・0011。
- docs/specs/http-server.md「設定と終了の契約」「検証との対応」、data-and-research.md と public-demo.md の収集・実行上限、ADR-0001〜0003、docs/development.md の検証・フック・本文検査。
- docs/use-cases.md、ADR-0013「決定と依存」「検証と制約」。

## 関連 Issue

Closes #56
