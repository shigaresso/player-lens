# PlayerLens

プレイヤーのレビューを収集・分析し、ゲーム企画に使える根拠付きレポートを生成するアプリケーションを開発する。

現在は Go の最小 HTTP サーバー、上限付きの Steam レビュー収集、JSON を出力するローカル CLI、ChatGPT へ収集結果を返す MCP、Docker Compose の開発・検証環境、Git・GitHub の規約検査を導入した。バックエンドの分析・レポート・DB は未実装とし、Web UI の公開デモとサンプルレポートはまだない。公開 MCP の基本接続先は [https://player-lens.onrender.com/tools](https://player-lens.onrender.com/tools) で、HTTP の初期化・一覧・不正入力の拒否を確認した。検証済み digest を配置する経路と実 ChatGPT 接続は未確認とする。

## 開発環境

Docker Engine と Docker Compose 2.23 以降（Docker Desktop でも可）を用意する。Go 1.27.1 はコンテナに含むため、ホストに Go をインストールする必要はない。規約検査には Python 3.12 以降と [開発手順のツール](docs/development.md#ローカルの準備) が必要となる。

```sh
python3 scripts/dev.py
```

8080 が使用中なら8081、8082…と順に試し、起動した URL を表示する。表示されたポートを使い、別のターミナルで HTTP 応答を確認する。

```sh
curl --fail http://localhost:8080/healthz
```

`ok` と改行を返す。ログは `docker compose logs --follow app` で確認できる。Compose Watch は `cmd/` と `internal/` の変更を同期してサーバーを再起動し、`go.mod`・`go.sum` の変更では依存を含めて再ビルドする。起動コマンドはイメージを1回ビルドし、前回の Watch 終了後の変更も含める。停止は Ctrl+C、コンテナとネットワークの片付けは `docker compose down` とする。Go のキャッシュ volume は保存される。

`.env.example` の `DEV_PORT` は探索を始めるポート（既定8080）、`PORT` はコンテナ内の待ち受けポートとなる。必要なら `.env` へコピーするか、次のように開始値を指定する。

```sh
DEV_PORT=3000 python3 scripts/dev.py
```

選択したポートは `.env` に保存しない。本番の `PORT` は自動変更しない。Docker のポート割り当て失敗だけを再試行し、設定・ビルド・healthcheck などの失敗は原因を表示して終了する。同じ Compose プロジェクトで起動済みの場合は、先の起動を Ctrl+C または `docker compose stop app` で停止してから起動する。`.env` は Git とビルド対象に含めない。

## レビュー収集 CLI

CLI 用イメージをビルドし、使用方法を確認する。

```sh
docker compose build collect
docker compose run --rm --no-deps -T collect --help
```

実 Steam API から収集するときは作品 ID と件数上限を指定する。次のコマンドは App ID 440 の日本語レビューを最大100件取得するため、外部通信が発生する。

```sh
docker compose run --rm --no-deps -T collect --app-id 440 --language japanese --max-reviews 100
```

標準出力へ収集条件・レビュー・終了理由を含む単一の JSON と改行を出す。help と診断は標準エラーへ出す。ビルドと実行を分け、JSON にビルドログを混ぜない。既定の取得上限は1000件、1リクエスト最大100件、最大100回・2分とする。件数・通信回数の上限による終了は全件取得を意味しない。

正常終了と help は終了コード0、不正入力は2、収集・出力の失敗は1とする。途中失敗では出力できた部分結果を残す。SIGINT・SIGTERM で収集をキャンセルし、部分結果の保存には独立した5秒の期限を渡す。通常の writer の実行中の書き込みは中断できず、原子的な出力・永続保存は保証しない。

収集と保存の操作は保存先の共通契約だけを使用し、起動部分で JSON の実装を注入する。JSON の変換は通信・保存から分離した。[CLI の仕様](docs/specs/review-cli.md) と [ADR-0006](docs/adr/0006-collection-output-boundary.md) を参照する。DB 保存、取得時刻・本文版・分析対象集合の管理、再開・再試行は未導入とする。通常テスト・CI は実 Steam API に接続しない。

## ChatGPT からレビューを取得する

同じサーバーの `/tools` で `collect_reviews` を利用できる。`python3 scripts/dev.py` が表示したポートを使い、ChatGPT デスクトップアプリへ `http://127.0.0.1:<ポート>/tools` を登録する。[MCP デバッグ手順](docs/mcp-debug.md) に設定と3件の手動確認例を記載した。生成 AI の API キーは不要で、分析は ChatGPT が担当する。登録済み MCP ツールをこの会話から呼び出し、App ID 4800590 の全言語3件を取得して、レビュー ID と分析の引用の対応を確認した。[実接続の記録](docs/worklogs/27-mcp-usage-limits.md) を参照する。

取得は既定20件・最大100件、最大5ページ、実行期限8秒とする。途中失敗は部分結果を返し、応答量超過時は件数を減らす案内を返す。レビューはメモリから返し、ファイルや DB へ保存しない。既定は localhost またはループバック IP の Host に限定する。`MCP_PUBLIC_URL` に HTTPS の `/tools` URL を指定すると、その公開 Host からも認証なしで利用できる。許可 Host と cross-origin 保護を維持し、OAuth は導入しない。[公開 MCP 仕様](docs/specs/mcp-public-access.md) と [ADR-0009](docs/adr/0009-public-mcp-access.md) を参照する。[MCP 仕様](docs/specs/review-mcp.md) と [ADR-0007](docs/adr/0007-review-mcp.md) を参照する。

収集開始を1分6回・24時間100回、同時2件まで受け付ける。`.env.example` の `MCP_REQUESTS_PER_MINUTE`・`MCP_REQUESTS_PER_DAY`・`MCP_MAX_CONCURRENT` で正の整数へ変更でき、不正な設定では起動を失敗させる。制限時は Steam へ接続せず MCP のエラー結果を返す。[利用制限仕様](docs/specs/mcp-usage-limits.md) と [ADR-0008](docs/adr/0008-mcp-usage-limits.md) を参照する。

枠は1プロセスのメモリ内で全利用者が共有する。再起動で回数が消え、複数台では各台に枠が存在する。固定窓の境界付近には受付が集中し得る。クラウドの請求額の上限は保証せず、公開構成の共有予算・使用量管理は別途決める。バックエンドで LLM を実行する分析機能と利用コード・残り回数の管理は未導入で、今回の MCP には含めない。

## 検証

```sh
bash scripts/check.sh
bash scripts/check-app.sh
python3 scripts/verify_hooks.py
```

アプリケーションの全検査は、Go の整形差分・静的検査・ユニットテスト、実 MCP クライアントとの結合テストを含む race 検査、本番用イメージのビルド・HTTP・MCP・設定・終了確認を行う。Docker の停止や検査失敗は成功扱いにしない。対象を分けて実行する場合は次を使う。

```sh
bash scripts/check-go.sh unit
bash scripts/check-lint.sh
bash scripts/check-go.sh integration
python3 scripts/check_image.py
```

テストは読み取り専用のソース共有を持つ Compose の検証サービスで実行する。整形の差分を修正するときは、同じ Go の development イメージで整形専用コンテナを起動する。

```sh
bash scripts/check-go.sh format
bash scripts/check-go.sh format-fix
```

検査はソースを書き換えず、`format-fix` だけが作業ツリーを修正する。pre-commit はステージ済み Go を整形検査し、pre-push と CI は固定版 golangci-lint も実行する。版・検査範囲・更新時の実ツール検証は [開発手順](docs/development.md#go-の整形修正とリンターの更新) に従う。

## 構成と制約

- `cmd/server/`：HTTP 応答、設定、サーバー起動・終了とテストを管理する。
- `cmd/collect-reviews/`：ローカル CLI の依存と出力・終了コードを組み立てる。
- `internal/steam/`：取得・収集と、保存先を注入する収集・保存操作を管理する。
- `internal/jsonoutput/`：JSON の純粋変換と writer への保存実装を管理する。
- `internal/reviewcli/`：CLI の入力検査と共通操作への接続を管理する。
- `internal/reviewmcp/`：MCP の入力・応答・HTTP の制限と共通収集操作への接続を管理する。
- `compose.yaml`：開発用サーバーと検証用 Go 環境を管理する。
- `Dockerfile`：development と production を分け、Go・実行用のベースイメージを digest で固定する。
- `scripts/`：開発用の起動と共通検査、Issue・PR・Release の作成手順を管理する。
- `docs/`：[仕様・ADR・開発手順](docs/README.md) を管理する。

`/healthz` は HTTP サーバーの応答を確認する。外部サービスや調査の成功を保証しない。[HTTP の仕様](docs/specs/http-server.md) に従い、PORT と終了時の期限を扱う。

本番用イメージの検査はソース共有やホストの Go に依存せず、一度ビルドした同じ image ID を使う。これはローカル・PR の動作検証であり、マージ後のリリース対象や Render に配置した成果物の検証とは区別する。Issue #31 で、main の CI 確認後に同じ image ID を検証して GHCR へ送る手動ワークフローと、digest 指定の Render Free 設定生成を追加した。この経路での実 registry 送信・配置は未実行である。[配置手順](docs/deployment.md) に従う。

Go の coverage 閾値・分岐網羅・mutation 検査は未導入とする。GitHub の新しいアプリケーション CI は push 後に対象 SHA の実行結果を確認する。ブランチ保護の制約は [開発手順](docs/development.md#必須チェック設定と残る限界) を参照する。

## 公開 MCP の接続確認

```sh
python3 scripts/check_public_mcp.py https://player-lens.onrender.com/tools
```

既定では実 Steam に接続せず、healthz と匿名 MCP の初期化・一覧・不正入力の拒否を確認する。ブラウザの GET `/tools` は405となる。実レビュー取得は明示オプションだけで行い、Free の停止・起動待ち、利用枠、実 ChatGPT 接続の確認は [配置手順](docs/deployment.md) を参照する。
