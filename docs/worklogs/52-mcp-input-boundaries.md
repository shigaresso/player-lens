# MCP 入力の JSON 読み取りと収集条件の構築を分離した

## 背景・目的

[Issue #52](https://github.com/shigaresso/player-lens/issues/52) に対応し、`parseInput` に同居していた入力形式の検査と収集条件の構築を分離した。

最新の origin/main（0d19c42527b8fc76dbca1cb86a805df6dea79e79）から `codex/52-mcp-input-boundaries` を作成した。以前のブランチは維持し、upstream は設定していない。push・PR 作成・Issue のクローズは行わず、ローカル確認用の状態にする。この文書は後続の PR 本文にも使用できる。

## 変更内容

- `tool_input.go` の `toolInput.UnmarshalJSON` が JSON の形式と正確な項目名を検査し、許可名を型の JSON タグから取得する構成にした。既定値や MCP 上限は読み取りで適用しない。
- `input_options.go` の `normalizeToolInput` が MCP の既定値・上限を適用し、評価種別の変換と共通正規化により収集条件を作る構成にした。
- `parseInput` は標準 json.Unmarshal と収集条件の構築を順に呼び、既存の正常結果と失敗時のゼロ値・`ErrInvalidArgument` を保持するエラーを返す境界にした。委譲だけになる decodeToolInput は残していない。
- 各処理の直接回帰テストを追加し、既存の `parseInput` の失敗テストで `errors.Is` による原因の識別も確認した。
- MCP 仕様とユースケースの文書へ責務の境界を記載した。
- 追加対応で、最初の `json.Unmarshal` と重複していた2回目の `Decode` による末尾検査と、不要になった `io` import を削除した。
- 先行する項目名の許可リストと重複する `DisallowUnknownFields()` の設定も、追加対応で削除した。

## 検証

### 抽出前の確認

`docker compose run --rm --no-deps -T verify go test -count=1 ./internal/reviewmcp` が成功した。既存の入力・評価種別・結果・利用制限を対象とし、抽出前の正常結果と失敗時の振る舞いを確認した。

### 抽出後の確認

抽出後にも同じ MCP ユニットテストが成功した。この段階では既存テストと期待結果は変更していない。動作を変えない抽出のため形式上の Red は作らず、責務分離と対応文書を 070e969 にまとめた。直接回帰テストは別の変更単位で追加した。

### 直接テストと原因の維持

同じ verify サービスで、責務ごとにテストを追加して関連テストを順に実行した。抽出済みの実装で最初から Green となり、テストに合わせたプロダクションコードの変更や、形式上の Red は作っていない。

- `go test -count=1 ./internal/reviewmcp -run '^TestDecodeToolInput$'`：23状況で、全項目と App ID の最大値、省略時のゼロ値、条件として不正でも形式上読める値の保持を確認した。未知・大文字の項目名、全6項目の null、型違い、型の範囲超過、壊れた JSON・追加値ではゼロ値と `ErrInvalidArgument` を返すことも確認した。途中まで読めた値を失敗時に返さないことを含む。
- `go test -count=1 ./internal/reviewmcp -run '^TestNormalizeToolInput$'`：15状況で、MCP の既定値・最小値・最大値、全評価／おすすめ／非おすすめの変換、言語とページ件数の共通正規化、MCP 上限超過と不正値の拒否を確認した。成功・失敗とも呼び出し側の入力を変更せず、不正時はゼロ値と識別可能な原因を返すことを確認した。
- `go test -count=1 ./internal/reviewmcp -run '^TestParseInput$'`：既存の不正入力テーブルで、非 nil の確認を `errors.Is(err, steam.ErrInvalidArgument)` へ強化して成功した。既存の正常値と失敗時のゼロ値の期待は維持した。

### 全体の検証

`bash scripts/check-go.sh format-fix` で固定版 Compose による整形を行った後、`bash scripts/check-app.sh` が成功した。整形・go vet・通常テスト全7パッケージ、integration タグを含む lint（0 issues）、Git 非依存3条件、race 付き通常・結合テスト全7パッケージを確認した。

実 SDK の対応4版の接続・一覧・呼び出し・入力エラー、共有利用枠、HTTP 切断と分析ガイドなどの既存結合テストも成功した。同じ本番用 image ID sha256:fd8e1899bc8da5da7ab42b2e5fa3e9682df04b7cc5ca62a889981cbf0c2d253d で、ローカル／公開設定の HTTP/MCP、Host・Origin 拒否、PORT、不正な公開 URL・利用制限の非0終了、SIGTERM/SIGINT 正常終了を確認した。実 Steam・実 LLM へは接続していない。

### フックと検証環境

070e969 の必須 pre-commit（規約ツール177テスト・静的検査・index の整形）と commit-msg が成功した。追加のコミットも必須フックを通常どおり実行する。検査ツール・Docker・CI・フック設定は、前回検証した 2a3da0a と今回の起点の差分がないことを確認した。同じ作業環境で前回成功した隔離フック全8テストの結果を再利用し、今回の再実行とは区別する。前回 PR #51 の規約 CI と lint CI でも、実フック・実検査ツールの独立検証が成功している。

push 前のため今回のコード CI と PR 作成 CLI の送信済み head/base を用いた検査は未実施。ローカルのコミット範囲・差分・PR 用本文を検査する。公開動作を変えない入力の責務整理のため、実 ChatGPT 接続と配置は今回実行していない。coverage の閾値・分岐・mutation（Stryker を含む）はリポジトリに未導入であり、既存検査の成功を品質目標の達成とは扱わない。

### AI のセルフレビュー

JSON の検査順序・toolInput の型とタグ・正確な項目名と null の検査・末尾の検査を維持し、後半の App ID・MCP 上限・既定値・評価種別・共通正規化も同じ順序で移したことを確認した。`parseInput` が両段階の失敗を既存と同じゼロ値・文言・原因へ変換することを確認した。`collect` の入力検査と利用枠の消費の順序、schema の内容、収集・CLI・依存モジュールには変更がない。人によるレビューは未実施。

抽出時点では、抽出前の JSON と条件構築の本文・定数が、戻り値の分離を除き移動先と一致することを読み取りスクリプトでも確認した。直接テストは実ソケット・時計・利用枠に依存せず、JSON 読み取りが意味の検査や既定値を先取りしない境界を確認する。直接テスト追加の段階では差分がテストと作業記録だけであることを確認した。

### 重複した末尾検査の削除（追加対応）

初回の抽出時に残した2回目の `decoder.Decode(new(any))` は、先にある `json.Unmarshal(data, &fields)` が入力全体を検査しているため不要だった。ユーザーの追加指示により、この再検査と `io` import を削除した。最初の Unmarshal と項目名・null・型の検査、正常結果と失敗時のゼロ値・原因は維持した。

テストを変更せず、`docker compose run --rm --no-deps -T verify go test -count=1 ./internal/reviewmcp -run '^Test(DecodeToolInput|ParseInput)$' -v` が成功した。特に TestDecodeToolInput の「後ろに追加の JSON」と TestParseInput の「JSONの後に値があった」の状況で、余計な値を拒否し、ゼロ値と `ErrInvalidArgument` を返すことを確認した。正常な単一オブジェクトの読み取りも引き続き成功した。動作を維持する削除のため、形式上の Red は作っていない。

削除後の `bash scripts/check-app.sh` も成功した。整形・vet・通常テスト全7パッケージ、lint（0 issues）、Git 非依存3条件、race 付き通常／結合テスト全7パッケージを確認した。本番用 image ID sha256:a1511b693d5b4f645e2b85e46750d304fb343ba1f7942aee06c2c538edfa92fa で HTTP/MCP・不正設定拒否・SIGTERM/SIGINT 正常終了も確認した。差分が重複検査・未使用 import の削除と対応文書だけであることを AI のセルフレビューで確認した。

### 重複した未知項目検査の削除（追加対応）

先行する `switch` が許可した項目名以外をすべて拒否するため、後続の `DisallowUnknownFields()` を削除した。大文字の表記を含めて正確な項目名を検査する責務は `switch` に残し、型と値の読み取りは既存の `Decode` が担当する。削除対象はこの設定1行だけとし、入力として許可する値とエラーの契約を維持した。

既存テストを変更せず、`docker compose run --rm --no-deps -T verify go test -count=1 ./internal/reviewmcp -run '^Test(DecodeToolInput|ParseInput|ParseInputReviewType)$' -v` が成功した。TestDecodeToolInput の未知／大文字の項目名、TestParseInput の未知項目・大文字の未定義／追加項目が、削除後もゼロ値と `ErrInvalidArgument` を返すことを確認した。正常入力・評価種別の既存テストも成功した。動作を維持する削除のため、形式上の Red は作っていない。

削除後の `bash scripts/check-app.sh` が成功し、整形・vet・通常テスト全7パッケージ、lint（0 issues）、Git 非依存3条件、race 付き通常／結合テスト全7パッケージを確認した。本番用 image ID sha256:45733e06c3739e89519cc8d3a6d5e1a0729664a04f279a1b0b7ce5f320330e82 で HTTP/MCP・不正設定拒否・SIGTERM/SIGINT 正常終了も確認した。設定1行の削除と対応文書だけが差分であり、先行する検査・型変換・利用枠・収集に変更がないことを AI のセルフレビューで確認した。

### JSON 読み取りの型への集約（追加対応）

ユーザーの追加指示により、JSON の読み取りを `toolInput.UnmarshalJSON` に集約した。許可名は標準 reflect で JSON タグから初期化時に一度だけ取得する。手書きの項目名一覧と `decodeToolInput` を削除し、`parseInput` は標準 `json.Unmarshal` を呼ぶ構成にした。`normalizeToolInput` と MCP の公開入力・schema・エラー・利用枠の契約は維持した。

TestToolInputJSONDecoding を先に追加し、標準 Unmarshal 経由では未知項目・大文字の項目名・null オブジェクト・null 項目の4状況がエラーにならず失敗する Red を確認した。コマンドは `docker compose run --rm --no-deps -T verify go test -count=1 ./internal/reviewmcp -run '^TestToolInputJSONDecoding$'`。実装後に同じ verify で `go test -count=1 ./internal/reviewmcp` が成功し、新しい読み取り契約と既存 MCP ユニットテストの Green を確認した。

既存23状況の読み取りテストは、対象 API の削除に伴いファイル・Test 名と呼び出し先をメソッドへ機械的に更新した。入力・期待値は変更していない。メソッドは一時値へ読み、成功時だけ受信先を更新する。標準 Unmarshal がメソッド呼び出し前に返す構文エラーも、parseInput の境界で従来どおりの原因へ変換する。

型への集約・標準 JSON 経由の5状況・既存テストの機械的な移動と対応文書を 190a55d にまとめた。その必須 pre-commit（規約ツール177テスト・静的検査・index の整形）と commit-msg が成功した。

続けて TestToolInputJSONReceiver を追加し、同じ verify で `go test -count=1 ./internal/reviewmcp -run '^TestToolInputJSONReceiver$'` が成功した。6状況で、成功時に省略項目を以前の値で補わないことと、未知項目・大文字・null・途中の型違いによる失敗時に既存の受信値を変更しないことを確認した。このテストは実装済みの契約で最初から Green となり、追加の実装変更や形式上の Red は作っていない。

型への集約後の `bash scripts/check-app.sh` が成功し、整形・vet・通常テスト全7パッケージ、lint（0 issues）、Git 非依存3条件、race 付き通常／結合テスト全7パッケージを確認した。本番用 image ID sha256:8f0fcc9264a4fe56504cad83ad299675e47e278e10b77a5c077fbdbd78315af9 で HTTP/MCP・不正設定拒否・SIGTERM/SIGINT 正常終了も確認した。

許可名は受信型のタグからだけ作り、初期化後は参照だけ行うことを確認した。既存の toolInput のフィールドとタグ、MCP の公開 schema、条件構築、収集操作・CLI・依存モジュールには変更がない。別の名前付き型へ読み取ることで UnmarshalJSON の再帰を避け、成功時だけ受信先を更新することを差分と標準 JSON 経由のテストで確認した。受信先テスト追加後の差分はテストと記録だけであることも AI のセルフレビューで確認した。

## 参照仕様・ADR

- docs/specs/review-mcp.md「接続と入力」「入力変換の境界」「結果・失敗・資源」「デバッグと検証」と ADR-0007「決定」
- docs/specs/collect-reviews.md「入力と正規化」、docs/specs/data-and-research.md の導入済み収集操作と ADR-0012「決定と境界」
- docs/specs/mcp-usage-limits.md「受付と固定窓」と ADR-0008「決定」
- docs/use-cases.md「パーツとの境界」
- docs/development.md のフック・アプリケーション検査・Issue／PR 作成手順と ADR-0001〜0003

## 関連 Issue

Closes #52
