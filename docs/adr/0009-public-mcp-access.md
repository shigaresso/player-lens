# ADR-0009：公開 MCP を認証なし・許可 Host と利用制限付きで提供する

## 状態

採用。Issue #29 の旧 OAuth 方針をユーザーとの議論に基づいて変更した。旧ローカル実装9コミットは `codex/29-public-mcp-auth` の `8e5c2c654a4e7ded23c039411822236a38b0bb32` に保護した。現ブランチへ OAuth を取り込まない。公開配置と実 ChatGPT 接続は未確認とする。

## 背景と比較

MCP は公開 Steam レビューの収集のみを担当し、クライアント側が分析する。OAuth による利用者管理は今回のポートフォリオで必要としない。LLM API キーをバックエンドで保持する分析機能は、別途、利用コード・残り回数・費用制限で扱う。

- OAuth は許可する利用者を制御できるが、ログイン・クライアント登録・トークン発行・鍵管理と外部サービスの運用が増えるため今回は採用しない。
- Host 検査を全て無効にする方式は、ローカル専用の安全な既定と公開先の境界を失うため採用しない。
- 明示的な公開 URL と許可 Host、既存の資源・実行制限による匿名公開を採用する。

## 決定

公開 URL が未指定なら従来の localhost 専用、HTTPS の `/tools` を指定した場合は認証なしで公開する。main で設定を検査し、HTTP adapter に注入する。許可 Host と Go の cross-origin 保護を受付前に適用する。SDK の localhost 保護は公開モードだけ独自検査へ置き換える。詳細は [公開 MCP 仕様](../specs/mcp-public-access.md) に記載する。

Issue #48 では、HTTP の受付判断を `mcpaccess` にまとめるため、ローカル用の判定を `ProtectLocal` へ分離する。`reviewmcp.NewHandler` は SDK と共有する利用枠を組み立て、設定に応じて `ProtectLocal` または `Access.Protect` を適用する。公開側とローカル側で既存の判定と検査順序を維持し、authority の正規化や nil 設定の意味は統一しない。保護単体を MCP SDK・収集依存から独立して検証できることを優先する。

ADR-0007 のローカル専用の適用範囲に公開モードを追加し、ADR-0008 の1プロセスの共有制限・既定値・解放責務は維持する。CLI と収集操作へ公開設定を混ぜず、空の Web adapter や将来用の DB・認証依存を追加しない。

## 利点と制約

認証サービスを契約せず、匿名クライアントから上限付きの収集を利用できる。Host は利用者の識別・認証ではなく、誰でも利用枠を消費できる。共有枠の占有や大量の拒否リクエストをこの変更だけで防いだとは扱わない。再起動・複数台の予算・クラウド請求上限は保証しない。実 URL の接続と資源の実測、公開インフラの制限は配置時に確認する。

Issue #56 では [ADR-0013](0013-domain-folder-layers.md) に従い、URL と Host 照合を mcpaccess/domain、HTTP の保護を mcpaccess/adapter に配置した。公開側は receiver の Access.Protect から Protect(access, next) へ変更し、nil・ゼロ値・Host・Origin の契約は維持する。

Issue #77 では HTTP の Protect／ProtectLocal を Controller と区別し、middleware として mcpaccess/adapter に維持する。MCP の生成・登録が外側から保護と Controller を接続する。Access と Host 照合、判定順と公開契約は変更しない。
