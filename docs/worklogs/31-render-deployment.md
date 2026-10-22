# Issue #31：Render 配置基盤

[Issue #31](https://github.com/shigaresso/player-lens/issues/31) に対応し、origin/main `30d69a7ef6cfc892857398f93168674e912bbb40` から `codex/31-render-deployment` を作成した。upstream は未設定。既存の未追跡 `.DS_Store` は保持してコミット対象から除いた。ユーザーの指示どおり push・PR 作成・Issue クローズは行っていない。

## 参照と範囲

文書一覧・公開デモ・公開 MCP・HTTP サーバー・MCP 利用制限の仕様、ADR-0001・0002・0009・0010、開発手順と公式 Render/GHCR/Docker 資料を参照した。[配置仕様](../specs/render-deployment.md) の「配布対象と事前検査」以降の契約を実装し、[ADR-0010](../adr/0010-render-deployment.md) の単一イメージ・手動公開・Free 構成を採用した。運用手順は [配置手順](../deployment.md) に記録した。

当初は Render サービス未作成の回答を受け、配置準備だけを対象とした。作業中にユーザーがサービスを作成したため、提示された URL の基本通信を追加で確認した。こちらからサービス作成・設定変更・再配置・秘密情報の登録は行っていない。

## 契約と Red・Green

同じ契約の入力違いをまとめ、テスト追加 → 意図した Red → 実装 → 関連テストの Green の順に確認した。失敗した状態をコミットしていない。

| コミット | 契約・確認した Red | Green の結果 |
| --- | --- | --- |
| `7a490e8` | 配置仕様・ADR の追加。文書のため Red は設けなかった | 参照と規約検査を確認した |
| `11500c5` | 既存 image ID の再利用。main の引数未対応による TypeError | 3テストで再ビルドなし・タグ拒否・失敗を確認した |
| `38b1003` | remote main・最新 push/main CI の事前検査。deploy モジュール未実装 | 成功・不一致・不完全な応答・ジョブ失敗の関連テストが成功した |
| `e3ac56a` | 一度のビルド・検証・送信と digest 照合。publish 未実装 | 順序・途中失敗・非再試行・manifest 不一致の関連テストが成功した |
| `afffbfe` | digest 固定の Free 構成。blueprint 未実装 | digest・公開 URL・資格情報名・初期設定の関連テストが成功した |
| `ef30670` | 公開プローブの契約と HTTP I/O を別サイクルで追加。モジュール未実装、続いて request_url/main 未実装 | 通常の非 Steam 通信・明示的な少量取得・異常応答・期限・量の上限の9テストが成功した |
| `11c8a3e` | 配置 CLI の接続。main/github_api 未実装 | deploy の15テストが成功した。手動ワークフローの静的検査も成功した |
| `3b4a8f1` | Content-Type の小文字ヘッダーで正常な healthz を拒否した | 公開プローブの10テストが成功した |
| `ff10ed3` | 利用枠の拒否・空の結果・別作品・上限違反の回帰テスト。既に Green のため失敗を作らなかった | 公開プローブの11テストが成功した。テストだけをコミットした |

通常の Python テストでは GitHub・registry・Render・Steam・LLM の通信を差し替えた。公開 CLI の失敗と、実サービスの利用制限を成功扱いにしないことを確認した。

## 全体検証

- `bash scripts/check-app.sh` が成功した。gofmt、vet、通常 Go テスト、golangci-lint（0 issues）、Git 情報の失敗・欠落・checkout 外の3条件、integration タグを含むテストと race detector、本番イメージを検証した。integration コマンドには通常のテストも含まれる。
- 本番イメージ `sha256:ad89d7e3dc99c5c8d4e482448e34249d1fe53591313f23717a37da739cb585e6` をビルドし、2つの PORT、SIGTERM/SIGINT、匿名 MCP、Host/Origin、metadata の404、不正な URL・PORT・利用制限設定を確認した。
- 同じ ID を `python3 scripts/check_image.py --image <上記 ID>` へ渡した検査も成功し、新しい入口では再ビルドしなかった。これはローカル実行環境での検証であり、将来の Ubuntu/linux/amd64 公開ジョブの実行結果ではない。
- `python3 scripts/verify_hooks.py` の8テストが成功した。隔離環境のフック検証と、本物の Docker を使ったアプリケーション検証を区別した。実 push は行っていない。
- 各コミットの必須 pre-commit・commit-msg が成功した。最終の Python 自動収集は174テスト。`.tools/actionlint -shellcheck=''`、差分の空白・ローカル文書リンクの確認が成功した。
- コード・差分を AI がセルフレビューした。人のレビューは未実施。Go の coverage 閾値・分岐網羅・mutation は未導入のままとし、Stryker は実行していない。

新しいテストだけの追加と文書更新では、成功済みの Go・Docker・実フックの全体検証を手動で重複実行していない。必須フックは各コミットで実行した。

## 公開 URL の確認（2026-10-23）

`https://player-lens.onrender.com/healthz` は200と `ok` を返した。初回の MCP 初期化・一覧はローカル専用の403だった。ユーザーの再デプロイ後、次のコマンドが成功した。

```sh
python3 scripts/check_public_mcp.py https://player-lens.onrender.com/tools
```

結果は protocol_version `2025-11-25`、steam_requested `false`、returned_count `0`、review_ids `[]` だった。匿名の初期化・ツール一覧・不正入力の拒否を確認した。ブラウザの GET `/tools` が405になることは異常ではなく、MCP の POST で確認する。

この操作で実 Steam・LLM は呼んでいない。実レビュー取得の `--collect-app-id` は未実行であり、ChatGPT アプリからの接続と分析の成功を示すものではない。配置済みサービスとローカルの image ID/digest の対応は未照合である。

## 実 CI の事前検査と残る作業

基点の main SHA に対する実 `deploy.py preflight` は「最新 CI が見つからないか応答が曖昧」で拒否した。GitHub の main の run 一覧は空、既存4ワークフローは active だった。原因は未特定であり、こちらで Actions の起動・設定変更を行っていない。判定不能を成功扱いにしなかったことを確認した。正常通過の契約は差し替えた API のテストで検証した。

このブランチを main に統合した後、同じ SHA の push/main CI の必要ジョブが成功してから手動公開する必要がある。registry への実送信・公開ジョブ・linux/amd64 イメージの実公開検査・Render 側の Blueprint 検証と digest 配置・ChatGPT の実接続・実レビュー取得・負荷実測は未実施である。無料枠の条件と private registry の認証は配置手順に記録し、課金・資格情報・公開リソースを変更していない。
