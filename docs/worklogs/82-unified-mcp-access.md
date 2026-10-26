# MCP の許可条件と HTTP 設定を分離し、保護手順を共通化した

## 背景・目的

[Issue #82](https://github.com/shigaresso/player-lens/issues/82) に対応した。不変の保護手順を共通化し、可変の許可条件・拒否文言・SDK 設定を生成時に一度だけ決める構成にした。ドメインの Host 判定と、HTTP 入口の設定の組み立てを分離した。

PR #81 の統合を確認し、origin/main の 68463bf4bc219000baa6519ab2bdc06d4acd04ba から codex/82-unified-mcp-access を作成した。先行するローカルコミットを保持し、最終案への訂正を追加した。upstream は設定せず、push・PR 作成・Issue のクローズは行っていない。

## 変更内容

- domain の Access は AllowsHost だけを持つ共通契約にした。NewLocal は状態を持たない localAccess、NewPublic は検査済み authority だけを持つ publicAccess を返す。両方とも非公開の値型とした。公開 URL の検査・Host 判定はこの境界に置いた。
- reviewmcp/adapter の NewHTTPConfig に、ローカル／公開の選択を1か所にまとめた。Access・固定の拒否文言・SDK オプションを HTTPConfig の非公開フィールドに保持し、利用側が別々に組み替える API は設けていない。共通の SDK オプションも1か所で定義した。
- NewHandler は生成済み HTTPConfig を受け取り、SDK オプションと Access・文言をそのまま使う。ゼロ値の設定は ErrInvalidConfig と nil handler を返す。Protect は Access・文言・next を受け取り、Host→Origin→next の共通手順を実行する。
- Access の IsPublic・HostRejectionMessage、両モードを選ぶ domain.New、ProtectLocal を残していない。Host の許可／拒否と入力エラーの if は、それぞれの判断として維持した。
- cmd/server/adapter の ReadMCPHTTPConfig で環境変数から設定を生成する。クライアント構築・待ち受けより前に不正設定を拒否する起動順序を維持した。NewPublic に直接空 URL を渡した場合は不正設定とし、暗黙にローカルへ切り替えない。
- [公開 MCP 仕様](../specs/mcp-public-access.md)、[MCP 仕様](../specs/review-mcp.md)、[受付一覧](../handler-routing.md)、[操作一覧](../use-cases.md)、ADR-0009・0013 を最終構成に整合させた。

## 検証

### 最終案の Red・Green と Refactor

NewHTTPConfig の生成契約のテストだけを先に追加し、NewHTTPConfig／HTTPConfig 未実装によるコンパイル失敗を確認した。両モードの生成・不正 URL の拒否を実装し、`go test -count=1 ./internal/reviewmcp/adapter -run '^TestNewHTTPConfig$'` の Green を確認した。

その後、呼び出し元・既存テストを新 API に接続し、旧 API を削除した。`go test -count=1 ./internal/mcpaccess/... ./internal/reviewmcp/adapter/... ./cmd/server/... ./internal` が成功した。HTTP 入出力の期待値は維持し、型・配置の整理のためだけの Red は追加していない。

同じ契約の入力違いはテーブル駆動テストにまとめた。生成テストでは、ローカル・公開・独自ポート・不正 URL の条件、固定文言、共通 SDK 設定、公開時の localhost 保護設定を確認した。未生成設定と nil 依存の拒否も既存の公開境界テストへ引き継いだ。

| 以前のテスト | 最終構成のテスト | 引き継いだ責務 |
| --- | --- | --- |
| TestNewAccess | TestNewPublicAccess | 公開 URL の検査と正常な生成。ローカル生成は Host 判定と HTTP 設定で確認した |
| TestAccessHostRejectionMessage | TestNewHTTPConfig | 拒否文言を HTTP 設定へ移し、許可条件・SDK 設定との組み合わせも確認した |
| TestReadMCPAccess | TestReadMCPHTTPConfig | 環境変数の読み取りと生成の成否を確認した |
| TestNewHandlerRejectsUninitializedAccess | TestNewHandlerRejectsZeroHTTPConfig | 未生成設定の構築エラー・nil handler・取得を行わない契約を確認した |

最終案の訂正前後で Test 関数154個を保持した。件数だけを保証とはせず、移管先と HTTP の既存期待値を照合した。ローカル AllowsHost、公開 AllowsHost と URL 解析・authority 正規化・照合の本体が、訂正前の実装と一致することも確認した。

### 先行する段階の記録

先行コミットでは、未指定時にも許可条件を生成する契約、ローカルの Host 判定の移管、単一 Protect で拒否形式を維持する契約、未生成設定を拒否する契約について、それぞれ意図した Red と関連テストの Green を確認した。その後、2つの実体型へ整理した。

さらに拒否文言を型が提供する段階では TestAccessHostRejectionMessage の未実装による Red と Green を確認した。この段階の Access は IsPublic・HostRejectionMessage を持っていたが、最終案では削除し、文言・SDK 設定の検証を HTTPConfig へ移した。先行段階の構成を現在の API として扱わない。

### 全体検証

- Docker の検証環境で `bash scripts/check-go.sh format-fix` が成功した。
- `bash scripts/check-app.sh` が成功した。gofmt・vet・全通常テスト、lint 0 issues、Git 非依存3条件、race 付き通常／結合テストを実行した。結合テストだけの成功とは扱わない。
- 同じ本番 image ID `sha256:fde51c5e879c3e59abfd16967f48dcc153fe3e8cd6a8ff442cbb86787eb748de` で、HTTP/MCP・公開／ローカルの Host/Origin・PORT・不正設定・SIGTERM/SIGINT を確認した。
- 実装コミット 35ec3d4 で必須フックが成功した。規約用の Python 177テスト・静的検査・整形検査・コミットメッセージ検査を通した。文書コミットのフックと最終のコミット範囲・差分・本文・リンクの結果は Issue に記録する。

### AI のセルフレビューと制約

生成側だけがモードを選び、利用側は設定済みの値を同じコードで使うことを確認した。非公開フィールドで設定の組み合わせを保護し、ゼロ値は構築時に拒否する。SDK オプションは HTTPConfig の値コピーを使い、呼び出し元によるオプションの書き換えを提供しない。

ローカル／公開それぞれの許可範囲、Host→Origin→next の順序、拒否文言と後続を呼ばない契約を維持した。生成に通信・listener・goroutine・時刻取得・利用枠の確保は追加していない。共有 limiter、Controller・UseCase、期限・キャンセル・応答変換までの枠保持と返却、healthz、SDK・依存の版は変更していない。人によるレビューは未実施。

GitHub 認証と Docker は権限付きで実行した。必須フック・対象・設定を省略／緩和していない。push 前のため pre-push・コード CI・送信済み head/base の PR 作成検査は未実施。実 Render 配置・実外部サービス接続、coverage 閾値・分岐・mutation（Stryker を含む）の導入・実行は行っていない。

## 参照仕様・ADR

- [公開 MCP 仕様](../specs/mcp-public-access.md) の起動設定・HTTP の受付・保護の組み立ての境界・検証
- [ADR-0009](../adr/0009-public-mcp-access.md) の許可条件と HTTP 設定・保護手順の分離、[ADR-0013](../adr/0013-domain-folder-layers.md) の domain と adapter の境界
- [MCP 仕様](../specs/review-mcp.md) の SDK 構築・公開保護・資源、[操作一覧](../use-cases.md) の責務
- [開発手順](../development.md) の Docker・通常／結合テスト・必須フック

## 関連 Issue

Closes #82
