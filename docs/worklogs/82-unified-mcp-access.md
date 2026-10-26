# ローカルと公開の MCP アクセス条件を Access に統一した

## 背景・目的

[Issue #82](https://github.com/shigaresso/player-lens/issues/82) に対応した。ユーザーが了承した、ローカル用にも Access を生成する構成を採用した。Host の許可条件を Access にまとめ、単一の Protect へ渡すことで、呼び出し側の保護関数選択をなくした。

PR #81 の統合を確認し、最新 origin/main の 68463bf4bc219000baa6519ab2bdc06d4acd04ba から codex/82-unified-mcp-access を作成した。既存ブランチとコミットを保持し、upstream は設定していない。push・PR 作成・Issue のクローズは行っていない。

## 変更内容

- mcpaccess.New は URL 未指定時も初期化済みのローカル用 Access を返す。非公開フィールドで生成済みの条件を保持し、IsInitialized・IsPublic・AllowsHost を提供した。nil とゼロ値は未初期化として拒否し、従来の nil＝ローカル設定という Go API 契約を変更した。
- ローカルの localHost を adapter から domain/local.go へ移し、その本体を維持した。公開用の URL 解析・authority 正規化・照合は既存の本体を維持した。両モードを同じ正規化へ変更せず、許可条件は domain、HTTP の実行手順は adapter に置いた。
- Protect は Access と next を受け取り、Host の検査→cross-origin 保護→next を1か所で実行する。モード別の403文言を構築時に選ぶ。ProtectLocal を削除した。
- NewHandler は常に Protect を呼び、SDK の DisableLocalhostProtection を Access.IsPublic から設定する。ローカルでは従来どおり SDK の localhost 保護を維持した。nil／ゼロ値 Access は ErrInvalidConfig を識別できる構築エラーとする。
- ReadMCPAccess の契約と呼び出し元・通常／結合テストを更新した。MCP の既存テスト12箇所では生成済み Access をテスト専用 helper から渡し、cmd/server の結合テスト1箇所でも New から生成した。
- [公開 MCP 仕様](../specs/mcp-public-access.md)、[MCP 仕様](../specs/review-mcp.md)、[受付一覧](../handler-routing.md)、[操作一覧](../use-cases.md)、ADR-0009・0013 を整合させた。新しい汎用 interface や framework は追加していない。

## 検証

### 契約ごとの Red・Green

Docker Compose verify で、テストだけの変更→意図した Red→関連する実装→Green の順に確認した。

| 契約 | 確認した Red | 実装と Green |
| --- | --- | --- |
| 未指定でも有効な Access を生成しモードを照会する | domain と設定読み取りのテストで IsPublic 未実装によるコンパイル失敗 | ローカル設定の生成と IsPublic・IsInitialized を実装し、両パッケージの通常テストが成功した |
| ローカルの Host 許可条件を Access が担当する | 従来のローカル判定では許可する入力を公開用の正規化で拒否した | localHost の本体を移し、AccessPolicy と domain・設定読み取りテストが成功した |
| 単一 Protect がローカルの拒否形式も維持する | 既存ローカルテストを Protect に接続し、拒否文言が公開用になって失敗した | 文言の選択を構築時に行い、mcpaccess の全通常テストが成功した |
| NewHandler は nil をローカル設定として解釈しない | nil Access で handler が正常に生成されることを検出した | SDK 設定を IsPublic に変更して常に Protect を呼び、nil／ゼロ値を含む関連テストが成功した |

主なコマンドは `go test -count=1 ./internal/mcpaccess/domain ./cmd/server/adapter`、`go test -count=1 ./internal/mcpaccess/adapter`、`go test -count=1 ./internal/reviewmcp/adapter -run '^TestNewHandler'`。最終の関連テストは `go test -count=1 ./internal/mcpaccess/... ./internal/reviewmcp/adapter/... ./cmd/server/... ./internal` で成功した。

同時に移行する必要がある生成・保護・利用側の API 契約を、対応するテストと1つの実装コミットにまとめた。既存の HTTP 入出力の期待値は維持し、Go の未指定・未初期化契約の期待値だけを合意した仕様へ変更した。

### 全体検証と内容保持

- `bash scripts/check-go.sh format-fix` が成功した。
- `bash scripts/check-app.sh` が成功した。gofmt・vet・全通常テスト、lint 0 issues、Git 非依存3条件、race 付き通常／結合テストを実行した。結合テストだけの成功とは扱わない。
- 同じ本番 image ID `sha256:2dfea8d87d31dc466b32678018c1c921f1504f4f38c87c9e60283d8ce2beae0a` で、HTTP/MCP・公開／ローカルの Host/Origin・PORT・不正設定・SIGTERM/SIGINT を確認した。
- localHost と公開側の parseHTTPSURL・canonicalAuthority・allowedAuthority の本体が基点と一致することを照合した。ローカル側と公開側で異なる判定を正規化で揃えてしまわないこともテストした。
- 既存の Test 関数152個を保持し、NewHandler の未初期化 Access を拒否するテスト1個を追加した。既存のローカル保護テストは Protect と生成済み Access へ接続し、全ケースの HTTP の期待値を維持した。
- 実装コミットの必須フックが成功した。文書コミットのフックと最終の差分・コミット範囲・本文・リンクの結果は Issue に記録した。

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
