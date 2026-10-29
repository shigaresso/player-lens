# Issue #105：生成・更新の境界と依存の契約

## 現在の採用範囲

ユーザーの追加指示により nilnil と NilAway はいったん外す。生成・更新境界の contractcheck、既存の errcheck・lint・テストは維持する。以下の導入時の評価・検証は削除前に実施した記録であり、現在の検査に含まれていることを意味しない。削除後の結果は末尾へ追記する。

## 対象と設計（当初の導入）

Issue: https://github.com/shigaresso/player-lens/issues/105

生成・更新位置を型情報で制限する検査、生成失敗の扱い、nil のフロー解析を組み合わせる。ADR-0019 に採用理由・対象と限界を記録した。今回は最初の対象型と NewHandler・CollectReviews の存在確認を整理し、外部入力や状態遷移の判断は維持する。

## Red・Green

- 直接生成の契約：型別名・別パッケージ・同名の別型・コピーの fixture を追加し、解析 API 未実装によるビルド失敗を確認した。型情報とパッケージ間の fact を使う実装後に成功した。
- 暗黙の生成の契約：未初期化の宣言・named result・new・省略フィールド・配列・make・変換・未確認の map/channel 取得で期待した診断が出ない Red を確認し、対応後に成功した。
- 更新境界の契約：フィールド変更・インクリメント・内部参照・ポインター経由の置き換え・nil への変更・公開フィールドを持つ宣言で Red を確認し、対応後に成功した。定義側のメソッドとコピーによる返却を許可した。

- 明示的な nil の契約：ポインターの nil 宣言・返却・型変換・コンテナーの要素とフィールドの初期化について診断欠落の Red を確認し、対応後に成功した。
- 内部参照の追加確認：関数引数、copy の更新先、append の戻り値、別表現へのポインター変換で迂回する例の Red を確認し、対応後に成功した。

## 解析ツールの評価

NilAway の既定設定では生成後のフィールドの nil 更新を検出しなかった。experimental-struct-init を有効にすると同じ違反 fixture を検出した。無名関数解析も評価したが、callback・goroutine 内で設定するテスト変数の誤検知が増えるため有効にしない。全 nil panic の排除を保証したとは扱わない。

integration を指定する -tags は、この解析ドライバーでは無効な互換オプションだった。GOFLAGS に -tags=integration を追加し、既存の -buildvcs=false を保持した。テストファイルの除外はしない。

実コードで、終了待ちの結合テストの共有 result 変数について診断された。既存の done チャネルで同期しており競合を発見したものではない。結果をチャネルで受け取る構成へ整理し、検査の除外や利用側の nil ガードを追加しなかった。

利用側のチェック削除後には、CLI の help 診断テストから nil の fetcher が渡る経路を検出した。help の分岐では使用しない既存テストだったが、常に有効な依存を渡す契約へ揃え、取得すると失敗するテスト用依存へ変更した。

生成・設定・状態・HTTP・収集の5パッケージの既存テストは移行前後とも成功した。失敗値のゼロ状態は reflect.Value.IsZero で全体を検証し、テスト側で新しい Limits{} を作成しない。未生成の設定や nil の依存を利用側で入力エラーにする旧テストは、契約変更に合わせて削除した。外部入力の AppID ゼロの拒否は維持した。

## 必須引数の遅延利用に対する追加確認

SDK callback に fetcher を保持する NewHandler では、通常の nil フローの推論だけでは nil の直接引数を検出しないことを隔離した実ソースのコピーで確認した。無名関数解析の有効化だけでもこの違反は検出しなかった。Go overlay による確認は解析用 AST へ変更が反映されなかったため、実ソースのコピーによる実行結果を採用した。

NewHandler と CollectReviews に NilAway の nonnil(fetcher) 注釈を追加し、必要な依存が存在する契約を呼び出し位置で明示した。SDK や callback の内部への推論だけに依存しない。NewHandler の nil と、生成後に nil にしたポインターを interface に渡すケース、CollectReviews の nil をいずれも検出した。実アプリケーションの一時コピーへの違反挿入を verify_go_tools.py に残し、3つの型の対象宣言と必須引数の注釈・設定が失われた場合も検査が失敗するようにした。作業ツリーや入力ファイルを書き換えない。

## 当初の導入時の検証（2ツールの削除前）

ローカルの Docker Compose、Go 1.27.1、linux/arm64 で確認した。

| コマンド・確認 | 結果 |
| --- | --- |
| 変更前後の5パッケージの go test -count=1 | 成功。設定生成・失敗の原因・コピーの分離・状態遷移・HTTP・収集を確認した |
| bash scripts/check.sh | Python 182件、空白・シェル構文・Lefthook・actionlint が成功した |
| bash scripts/check-app.sh | 全パッケージの gofmt・vet・ユニットテスト、共通 lint、結合テストを含む race、本番用イメージが成功した |
| 共通 lint | アプリケーションと独立モジュールの lint がともに0件、contractcheck・NilAway と検査自身のテストが成功した |
| python3 scripts/verify_go_tools.py | 実 nilnil・errcheck・NilAway・contractcheck の正常・違反と入力保持、実ソースへの違反挿入、ステージ済み整形と部分ステージ保持が成功した |
| python3 scripts/verify_hooks.py | 隔離した Git/Lefthook の11件が成功した。Docker の接続・失敗伝播はテスト用実装で確認し、実アプリケーションの検証とは区別した |
| 文書と差分 | ローカルリンク186件（アンカーを除く）と git diff --check を確認した。端末のホーム絶対パスを変更内容へ含めなかった |

最終の本番用 image ID は `sha256:15039eeea9de186deecc317ea904c9fd4d237dee9321aed15f6babc7bced4f06`。同じイメージで PORT=8080/10000、HTTP・MCP、ローカル・公開の保護、不正な PORT・MCP 公開 URL・3つの上限設定、SIGINT/SIGTERM の正常終了を確認した。本番の依存へ検査モジュールを追加せず、検証用ステージにだけ解析バイナリを配置した。

## セルフレビューと残る範囲

Limits の失敗値は NewLimits の結果を伝播し、数値解析の原因と優先順を維持した。HTTPConfig の SDK オプションへの参照は定義ファイル内の newHandler に収めた。CollectionState の更新ロジック、同時枠・期限・Body と接続プールの解放、外部入力と応答の検査は変えていない。生成に失敗した値や nil の依存を利用側で拒否する旧契約だけを、生成側と静的検査の事前条件へ移した。

対象は明示した3型から始め、ゼロ値が有効な型や未検証の入力 DTO は一律禁止しない。配置と nil の検査だけで、更新内容の業務不変条件や全 nil panic を完全証明したとは扱わない。通常の go test ./... は独立モジュールを含まないため、共通 lint 内で検査モジュールの vet・テスト・lint を別に実行した。

実 Steam・LLM・Render・GitHub CI は実行していない。coverage 閾値・分岐・mutation は現在未導入であり、Stryker を実行したとは記載しない。push・PR 作成・Issue のクローズは行わず、検査基盤とアプリケーションの契約移行をローカルコミットに分ける。

## nilnil・NilAway の削除

ユーザーの追加指示を Issue #105 に反映し、nilnil の有効化・設定と専用 fixture、NilAway の実行・ビルド・依存と不要な間接依存、nonnil(fetcher) 注釈と専用の違反 probe を削除した。golangci-lint の固定した配布バイナリ自体は維持するため、同梱された未使用の nilnil 実装を取り除く独自ビルドは行わない。有効な検査から外した。

contractcheck の実装・対象3型・ユニットテスト・実 CLI の正常/拒否/入力保持と実ソースの生成違反の検証を維持した。生成成功した依存を渡す事前条件と、利用側の存在確認を繰り返さない方針は維持し、存在確認の再追加や新しい自作禁止ルールの追加は行っていない。一般的な nil の流れ、nil と成功の同時返却、利用側の引数の nil 再検査を機械的に禁止する専用ルールは未導入である。

削除後もローカルの Docker Compose、Go 1.27.1、linux/arm64 で確認した。振る舞いを増やさない削除のため、形式だけの Red は追加しなかった。

| コマンド・確認 | 結果 |
| --- | --- |
| bash scripts/check.sh | Python 182件、空白・シェル構文・Lefthook・actionlint が成功した |
| bash scripts/check-app.sh | 全パッケージの gofmt・vet・ユニットテスト、共通 lint、結合テストを含む race、本番用イメージが成功した。アプリケーションと検査モジュールの lint はともに0件だった |
| python3 scripts/verify_go_tools.py | 既存 lint、contractcheck の正常・拒否・入力保持、実ソースの3型の違反、ステージ済み整形・部分ステージ保持・明示的な整形修正を確認した |
| 削除と残存ツール | 実行・ビルド・依存・注釈の参照が残っていないことを検索した。検証用イメージで contractcheck が存在し、NilAway が存在せず、nilnil が有効な lint に含まれないことを確認した。独立モジュールを tidy し、不要な依存を削除した |
| 文書と差分 | 変更した Markdown のローカルリンク133件（アンカーを除く）、開発手順の変更したアンカー、git diff --check を確認した。端末のホーム絶対パスを変更内容へ含めなかった |

本番用 image ID は `sha256:75cfb3c6cb3e3de91a18b67f9822733c5a7a2f9f65f04ddb977f741e365b8ac2`。同じイメージで PORT=8080/10000、HTTP・MCP、ローカル・公開の保護、不正設定の非0終了と公開 URL の非表示、SIGINT/SIGTERM の正常終了を確認した。contractcheck の実装・テストは変更していない。必須 pre-commit・commit-msg はローカルコミット時にも実行する。GitHub CI・実 Steam・LLM・Render は未実行であり、push・PR 作成・Issue のクローズは行わない。
