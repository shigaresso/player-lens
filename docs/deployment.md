# Render への配置手順

## 現在の状態

Issue #31 は配置準備であり、このブランチから Git/registry へ push していない。手動公開ワークフロー・GHCR 送信・digest を指定した Render 配置は未実行である。

作業中にユーザーが Render サービスを作成し、`https://player-lens.onrender.com/healthz` の200を確認した。初回の MCP はローカル専用の403だったが、ユーザーの再デプロイ後に `https://player-lens.onrender.com/tools` で匿名の initialize・tools/list・不正入力の拒否を確認した。実レビュー取得と ChatGPT アプリでの接続は未確認である。このサービスの image ID/digest と配置経路は照合していない。

## main へマージした後のイメージ公開

1. この PR を main に統合し、同じ SHA の push/main の規約検査とアプリケーション検査が成功したことを確認する。PR の CI で代用しない。
2. GitHub Actions の「配置用イメージ公開」を main から手動実行する。main 以外では公開ジョブは実行されない。
3. ワークフローは linux/amd64 を一度ビルドし、得た image ID を `check_image.py --image` で再ビルドせず検証する。成功後だけ GHCR へ送信し、manifest の config digest と image ID の一致を確認する。
4. Actions の Summary に記録した target SHA・image ID・`ghcr.io/...@sha256:...`・生成した Render 設定を保存する。`latest` タグや配置時の再ビルドへ置き換えない。

事前検査だけを実行する場合は、対象 main の40桁 SHA を `DEPLOY_TARGET_SHA` に設定したうえで次を使う。

```sh
python3 scripts/deploy.py preflight --repo shigaresso/player-lens --target "$DEPLOY_TARGET_SHA"
```

作業時の main では既存ワークフローは active だが実行記録が未検出であり、この CLI が拒否した。記録を削除したり、PR/手動実行の成功だけで置き換えたりせず、push/main の必要ジョブが成功した状態を用意する。確認不能を通過させる例外は導入していない。

## GHCR の資格情報と無料利用

手動公開ジョブは `GITHUB_TOKEN` の packages:write で送信し、ジョブ終了時に Docker のログイン情報を破棄する。既存 CI の権限は増やさない。package の visibility を自動で public に変更しない。

GHCR の package が private の場合、Render の Workspace Settings の registry credentials に読み取り用資格情報を登録する。GitHub の PAT は read:packages と対象 package を読める権限を持つものとし、Render へ直接登録する。チャット・Git・設定ファイルへトークンを貼らない。生成設定に記録するのは資格情報の名前だけである。public package の匿名 pull ができることを確認済みの場合だけ資格情報名を省略できる。

Render Free は0.1 CPU/512 MB、idle 15分で停止し、起動に約1分かかる。750時間/月・帯域・外向き通信などの制約があり、メモリ内の実行枠も再起動で消える。Hobby ワークスペースの選択と実行サービスの料金は別である。支払方法を登録しない運用では無料枠超過時の停止を許容する。GitHub Actions の実行枠・GHCR の利用条件も実行前に確認し、無料で無制限とは扱わない。

2026-10-23に確認した公式資料：[Render Free](https://render.com/docs/free)、[Blueprint の plan](https://render.com/docs/blueprint-spec)、[GHCR](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)。

## Render 設定の生成と適用

Actions の Summary にある `publication.json` の内容をローカルに保存する。以下の例の `ghcr-reader` は、Render に実際に登録した資格情報名に置き換える。保存する内容は token ではない。

```sh
python3 scripts/deploy.py blueprint --repo shigaresso/player-lens \
  --receipt publication.json --output render.generated.yaml \
  --public-url https://player-lens.onrender.com/tools --credential ghcr-reader
```

生成物は JSON 形式の有効な YAML であり、Python 標準ライブラリだけで生成する。出力ファイルは新規作成だけを許可し、既存ファイルを上書きしない。生成物は Git 管理外とし、実際の digest なしに適用可能な架空の設定を作らない。

生成する構成は `runtime: image`・`plan: free`・1インスタンス・`healthCheckPath: /healthz`・PORT 10000、既存の6回/分・100回/24時間・同時2件である。DB・ディスク・有料サービス・LLM API キーを追加しない。

Render の Existing Image で image-backed サービスを作成し、Summary の厳密な digest と必要な registry credential を指定する。Blueprint を使う場合は生成設定を読み、Render 側でも検証してから適用する。既存サービスへ適用する場合は、名前・runtime・URL・資格情報が意図した対象か確認する。Git-backed の再ビルド構成の成功を、この経路の成功と扱わない。

初回作成で公開 URL が未定なら `--public-url` を省略して生成する。この場合 MCP はローカル専用で起動し、healthz だけが確認できる。サービスの実 URL 確定後に `MCP_PUBLIC_URL` を HTTPS の `/tools` へ設定し、同じ digest で再デプロイする。URL が変わった場合も同様に更新する。

既存の公開 URL を使うときは、GitHub の repository variable `MCP_PUBLIC_URL` に `https://player-lens.onrender.com/tools`、`RENDER_REGISTRY_CREDENTIAL` に実際の資格情報名を設定すれば、手動ワークフローが生成設定へ反映する。変数の未設定時はローカル専用・資格情報なしの設定になる。これらの変数の設定は今回行っていない。

## 公開後の確認

```sh
python3 scripts/check_public_mcp.py https://player-lens.onrender.com/tools
```

既定では healthz・匿名初期化・一覧・不正入力だけを検査し、実 Steam を呼ばない。ブラウザの GET `/tools` が405でも、MCP の POST が成功するかで判断する。403でローカル Host のみと返る場合は公開 URL の設定と配置バージョンを確認する。

実レビューを確認するときだけ、明示的に少量取得する。

```sh
python3 scripts/check_public_mcp.py https://player-lens.onrender.com/tools --collect-app-id 4800590
```

この操作は実 Steam にアクセスし、開始回数を消費する。1件・1ページだけを要求し、取得件数とレビュー ID を表示する。本文は出力しない。0件・利用制限・失敗を取得成功として扱わない。今回、この明示オプションは実行していない。

各 HTTP 要求は15秒・応答256 KiBまでで、リダイレクト・自動再試行・自動ウォームアップを行わない。Free が停止中の場合は手動で起動を待って再実行する。常時起動させるための定期アクセスは追加しない。

最後に実クライアントから匿名接続・収集・分析を確認する。SDK/HTTP の検査と ChatGPT アプリの実接続を区別して記録する。Render のメモリと応答時間・エラーを観測し、実測前に同時上限を3へ増やさない。

イメージを更新・ロールバックするときも検証済みの digest を明示し、旧 digest を registry に保持する。送信・照合失敗時は自動再試行せず、registry の状態を確認する。Render への実配置と URL の確認結果は別途記録する。

[配置仕様](specs/render-deployment.md)、[ADR-0010](adr/0010-render-deployment.md)、[作業記録](worklogs/31-render-deployment.md) を参照する。
