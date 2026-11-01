# Render 配置基盤

## 目的と状態

Issue #31 で配置準備を実装する。当初は Render サービス未作成との回答を受けたが、作業中にユーザーが作成した。今回のローカル作業では Git/registry への push・公開リソース作成・実配置を行わない。実 CI・registry への送信・検証済み digest の実配置・ChatGPT 接続・負荷実測はマージ後の確認事項である。ユーザー配置の `https://player-lens.onrender.com/tools` では HTTP の匿名初期化・一覧・不正入力の拒否を確認した。

## 配布対象と事前検査

- 対象は40桁の SHA と OWNER/REPO を明示する。checkout の HEAD と clean な状態、remote main の先端が一致することを要求する。
- 同じ SHA の最新 push/main の `conventions.yml` の verify、`application.yml` の frontend-unit/go-lint/go-unit/go-integration/production-image が一意に存在し、全て completed/success であることを要求する。未完了・欠落・重複・取得不完全・通信失敗では公開しない。
- `linux/amd64` の production イメージを一度だけビルドする。得た厳密な image ID を既存の本番検査へ渡し、再ビルドせず HTTP/MCP・設定・終了を検証する。PR の成果物で代用しない。
- 検証後に main の先端を再確認し、同じ image ID を GHCR の SHA タグへ送る。送信後の manifest digest で読み取り、config.digest とローカル image ID の一致を要求する。返す配置先はタグではなく `ghcr.io/owner/repo@sha256:...` とする。
- registry の visibility は自動変更しない。書き込みと照合に失敗した場合は自動再試行せず、registry に送信済みの可能性を報告する。main と registry の操作は原子的ではない。main が更新された場合に別ソースへ自動置換しない。

## Render の構成

- 生成する Blueprint は image-backed の Web Service、Free、1インスタンス、`/healthz` とする。Dockerfile を Render で再ビルドしない。region は初期値 oregon とする。
- 入力イメージはこのリポジトリの GHCR manifest digest だけを受け付ける。タグ、別 registry、別リポジトリは拒否する。
- `PORT=10000` と既存の開始回数・同時数（6/分・100/24時間・同時2件）を明示する。DB・ディスク・LLM API キー・OAuth を追加しない。
- 初回作成で URL が未定なら `MCP_PUBLIC_URL` を空にしてローカル専用で起動する。healthz の成功を公開 MCP の接続成功と扱わない。実 URL 確定後に HTTPS の `/tools` を明示し、同じ digest の設定更新で公開する。
- private GHCR は Render 側へ read:packages の registry credential を登録し、設定には資格情報名だけを参照する。トークンを Git/Blueprint/イメージへ書かない。
- JSON 形式も YAML として読み取れるため、標準ライブラリで Blueprint を JSON 形式の YAML として生成する。架空の digest/実 URL を持つ配置可能な設定をコミットせず、実際の digest から生成した `render.generated.yaml` は Git 管理外とする。

## 公開接続の検査

HTTPS の `/tools` URL だけを受け付け、認証なしで healthz・initialize・tools/list・不正入力の tools/call を確認する。要求/応答には期限とサイズ上限を設け、リダイレクト・JSON-RPC の別 ID・error・不正な結果はエラーとする。再試行や自動ウォームアップを行わない。

既定では実 Steam を呼ばない。明示的な `--collect-app-id` の時だけ1件・1ページを取得し、取得件数・IDを検証する。レビュー本文は出力せず、実レビューの0件や上限拒否は成功として報告しない。通常テストは HTTP 応答を差し替え、外部サービスへ接続しない。

## 無料枠と運用の制約

Free は idle 15分で停止し、次の起動には約1分かかる。ローカルファイルは再起動/再配置/停止で失われる。750時間/月・帯域・外向き通信などの制約があり、Hobby ワークスペースとサービスの実行料金は別である。支払方法を登録しない運用では超過時の停止を許容し、有料プランへ自動変更しない。無料で無制限に利用できるとは記載しない。

GHCR と GitHub Actions の料金・利用枠も配置時に再確認する。新しいワークフローは main からの手動起動のみとし、既存 PR/CI の再実行や自動デプロイを追加しない。公開後にメモリ・応答時間・拒否を測り、実測前に同時数を3へ増やさない。1プロセスのメモリ制限・請求上限の保証範囲は従来どおりである。

## 検証と参照

Python の通常テストは subprocess/HTTP を差し替え、成功・失敗・書き込み前の拒否・再ビルドなし・非再試行を確認する。固定版のアプリ全体検査と実フックを確認し、実資源の状態を模擬したテストと実配置を混同しない。

[ADR-0010](../adr/0010-render-deployment.md)、[ADR-0001](../adr/0001-runtime-and-deployment.md)、[公開 MCP](mcp-public-access.md)、[作業記録](../worklogs/31-render-deployment.md) に対応する。
