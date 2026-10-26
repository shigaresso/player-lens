# MCP の許可条件を型で分け、Access と Protect を共通化した

## 背景・目的

[Issue #82](https://github.com/shigaresso/player-lens/issues/82) に対応した。ユーザーが了承した、ローカル用にも Access を生成する構成を採用した。Host の許可条件を Access の共通契約で扱い、単一の Protect へ渡すことで、呼び出し側の保護関数選択をなくした。最初の構造体による実装後、ユーザーの指摘に基づきローカル用と公開用を2つの実体型に分けた。先行するローカルコミットは保持した。

PR #81 の統合を確認し、最新 origin/main の 68463bf4bc219000baa6519ab2bdc06d4acd04ba から codex/82-unified-mcp-access を作成した。既存ブランチとコミットを保持し、upstream は設定していない。push・PR 作成・Issue のクローズは行っていない。

## 変更内容

- Access は AllowsHost・IsPublic・HostRejectionMessage を持つ共通契約にした。mcpaccess.New は未指定時にフィールドを持たない localAccess、指定時に検査済み authority だけを保持する publicAccess を、非公開の値型として返す。localOnly フラグと IsInitialized を廃止した。Access のゼロ値は nil とし、Protect と NewHandler の公開境界で未生成として拒否する。従来の nil＝ローカル設定という Go API 契約を変更した。
- ローカルの判定を domain/local.go の localAccess.AllowsHost へ移し、その本体を維持した。公開用は domain/public.go に分け、URL 解析・authority 正規化・照合の本体を維持した。両モードを同じ正規化へ変更せず、許可条件は domain、HTTP の実行手順は adapter に置いた。
- Protect は Access と next を受け取り、Host の検査→cross-origin 保護→next を1か所で実行する。型が提供する固定の拒否理由を構築時に取得し、Protect 自身はモードで分岐しない。ProtectLocal を削除した。
- NewHandler は常に Protect を呼び、SDK の DisableLocalhostProtection を Access.IsPublic から設定する。ローカルでは従来どおり SDK の localhost 保護を維持した。nil／ゼロ値 Access は ErrInvalidConfig を識別できる構築エラーとする。
- ReadMCPAccess の契約と呼び出し元・通常／結合テストを更新した。MCP の既存テスト12箇所では生成済み Access をテスト専用 helper から渡し、cmd/server の結合テスト1箇所でも New から生成した。
- [公開 MCP 仕様](../specs/mcp-public-access.md)、[MCP 仕様](../specs/review-mcp.md)、[受付一覧](../handler-routing.md)、[操作一覧](../use-cases.md)、ADR-0009・0013 を整合させた。共通契約は実在する Host 判定・拒否理由と SDK 設定に必要なものに限定し、汎用 framework は追加していない。

## 検証

### 最初の構造体による実装の Red・Green

以下は最初の構造体による実装の記録である。Docker Compose verify で、テストだけの変更→意図した Red→関連する実装→Green の順に確認した。

| 契約 | 確認した Red | 実装と Green |
| --- | --- | --- |
| 未指定でも有効な Access を生成しモードを照会する | domain と設定読み取りのテストで IsPublic 未実装によるコンパイル失敗 | ローカル設定の生成と IsPublic・IsInitialized を実装し、両パッケージの通常テストが成功した |
| ローカルの Host 許可条件を Access が担当する | 従来のローカル判定では許可する入力を公開用の正規化で拒否した | localHost の本体を移し、AccessPolicy と domain・設定読み取りテストが成功した |
| 単一 Protect がローカルの拒否形式も維持する | 既存ローカルテストを Protect に接続し、拒否文言が公開用になって失敗した | 文言の選択を構築時に行い、mcpaccess の全通常テストが成功した |
| NewHandler は nil をローカル設定として解釈しない | nil Access で handler が正常に生成されることを検出した | SDK 設定を IsPublic に変更して常に Protect を呼び、nil／ゼロ値を含む関連テストが成功した |

主なコマンドは `go test -count=1 ./internal/mcpaccess/domain ./cmd/server/adapter`、`go test -count=1 ./internal/mcpaccess/adapter`、`go test -count=1 ./internal/reviewmcp/adapter -run '^TestNewHandler'`。最終の関連テストは `go test -count=1 ./internal/mcpaccess/... ./internal/reviewmcp/adapter/... ./cmd/server/... ./internal` で成功した。

同時に移行する必要がある生成・保護・利用側の API 契約を、対応するテストと1つの実装コミットにまとめた。既存の HTTP 入出力の期待値は維持し、Go の未指定・未初期化契約の期待値だけを合意した仕様へ変更した。

### 2種類の実体型への整理

生成・許可条件・HTTP 応答の動作を変えない Refactor として整理した。生成と受け渡しを Access の共通契約に変更し、ローカル用・公開用の具体型は外部から直接組み立てられない形にした。正常な生成結果は値型なので nil ポインターを含まない。

IsInitialized の照会と構造体のゼロ値の検査を廃止した。interface のゼロ値は nil のため、重複する未生成ケースを Protect と NewHandler の公開境界の拒否へまとめ、生成済みの Host 判定では IsPublic・AllowsHost を確認する。HTTP の入力・期待値は維持し、内部フィールドをなぞる検査や形のためだけの Red は追加していない。関連パッケージの同じ通常テストが Green になった。

### 利用側でのモード再判定の削除

型を分けた後も Protect に IsPublic で拒否文言を選ぶ分岐が残っていたため、生成側で選ばれた型が固定の拒否理由を提供する HostRejectionMessage を追加した。HTTP ステータス・ヘッダー・本文形式の責務は adapter に維持した。

TestAccessHostRejectionMessage を先に追加し、未実装メソッドによるコンパイル失敗の Red を確認した。両型が既存と同じ固定理由を返し、Protect がその値を使う実装で関連パッケージの Green を確認した。Protect と NewHandler のソースに IsPublic を使ってモードを選び直す if／switch がないことも確認した。NewHandler の SDK 設定は IsPublic の値をそのまま使う。Host の許可／拒否や入力エラーの条件分岐は、それぞれの判断として維持する。

### 全体検証と内容保持

- `bash scripts/check-go.sh format-fix` が成功した。
- `bash scripts/check-app.sh` が成功した。gofmt・vet・全通常テスト、lint 0 issues、Git 非依存3条件、race 付き通常／結合テストを実行した。結合テストだけの成功とは扱わない。
- 同じ本番 image ID `sha256:dae22536c9c91cbb8b6fbbc88e782d37fb7b8acc101e8f6b9eaa43807aae6d3a` で、HTTP/MCP・公開／ローカルの Host/Origin・PORT・不正設定・SIGTERM/SIGINT を確認した。
- ローカルの AllowsHost と公開側の parseHTTPSURL・canonicalAuthority・allowedAuthority の本体が、型を分ける前と一致することを照合した。ローカル側と公開側で異なる判定を正規化で揃えてしまわないこともテストした。
- 最初の実装で既存の Test 関数152個を保持し、NewHandler の未生成 Access を拒否するテスト1個を追加した。型を分ける整理でも153個の Test 関数を保持した。拒否理由の契約テスト1個を追加し、最終構成は154個の Test 関数を保持している。既存のローカル保護テストは Protect と生成済み Access へ接続し、全ケースの HTTP の期待値を維持した。
- 実装・型を分ける整理・モード再判定の削除のコミットで必須フックが成功した。文書コミットのフックと最終の差分・コミット範囲・本文・リンクの結果は Issue に記録した。

### AI のセルフレビューと制約

URL 未指定でも条件が生成され、ローカル／公開の両方で Protect が呼ばれること、公開時だけ SDK の localhost 保護を無効にすること、Host→Origin→next の順序と拒否文言を確認した。nil／ゼロ値・nil next は構築エラーとし、エラーが handler とともに成功扱いで返らないことを確認した。

構築に新しい listener・接続・goroutine・再試行・ログ出力を追加していない。共有 limiter、Controller・UseCase、期限・キャンセル・応答変換までの枠保持と返却、healthz、SDK・依存の版は変更していない。人によるレビューは未実施。

GitHub 認証と Docker は権限付きで実行した。必須フック・対象・設定を省略／緩和していない。push 前のため pre-push・コード CI・送信済み head/base の PR 作成検査は未実施。実 Render 配置・実外部サービス接続、coverage 閾値・分岐・mutation（Stryker を含む）の導入・実行は行っていない。

## 参照仕様・ADR

- [公開 MCP 仕様](../specs/mcp-public-access.md) の起動設定・HTTP の受付・保護の組み立ての境界・検証
- [ADR-0009](../adr/0009-public-mcp-access.md) の許可条件と保護手順の統一、[ADR-0013](../adr/0013-domain-folder-layers.md) の domain と adapter の境界
- [MCP 仕様](../specs/review-mcp.md) の SDK 構築・公開保護・資源、[操作一覧](../use-cases.md) の責務
- [開発手順](../development.md) の Docker・通常／結合テスト・必須フック

## 関連 Issue

Closes #82
