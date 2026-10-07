# PlayerLens

プレイヤーのレビューを収集・分析し、ゲーム企画に使える根拠付きレポートを生成するアプリケーションを開発する。

現在は Go の最小 HTTP サーバー、Docker Compose の開発・検証環境、Git・GitHub の規約検査を導入した。レビュー収集・分析・レポート・DB・MCP は未実装とし、公開デモ URL とサンプルレポートはまだない。

## 開発環境

Docker Engine と Docker Compose 2.23 以降（Docker Desktop でも可）を用意する。Go 1.27.1 はコンテナに含むため、ホストに Go をインストールする必要はない。規約検査には Python 3.12 以降と [開発手順のツール](docs/development.md#ローカルの準備) が必要となる。

```sh
docker compose up --build --watch app
```

別のターミナルで HTTP 応答を確認する。

```sh
curl --fail http://localhost:8080/healthz
```

`ok` と改行を返す。Compose Watch は `cmd/` と `go.mod` の変更を同期し、ビルド後にサーバーを再起動する。起動時も `--build` を使い、前回の Watch 終了後の変更を含める。停止は Ctrl+C、コンテナとネットワークの片付けは `docker compose down` とする。Go のキャッシュ volume は保存される。

`.env.example` に PORT とローカル公開ポートの説明を記載した。必要なら `.env` へコピーして変更する。`.env` は Git とビルド対象に含めない。

## 検証

```sh
bash scripts/check.sh
bash scripts/check-app.sh
python3 scripts/verify_hooks.py
```

アプリケーションの全検査は、Go の整形差分・静的検査・ユニットテスト、結合テストを含む race 検査、本番用イメージのビルド・HTTP・設定・終了確認を行う。Docker の停止や検査失敗は成功扱いにしない。対象を分けて実行する場合は次を使う。

```sh
bash scripts/check-go.sh unit
bash scripts/check-go.sh integration
python3 scripts/check_image.py
```

テストは読み取り専用のソース共有を持つ Compose の検証サービスで実行する。整形の差分を修正するときは、同じ Go の development イメージで整形専用コンテナを起動する。

```sh
docker compose build verify
docker run --rm -v "$PWD:/src:rw" -w /src player-lens-verify gofmt -w .
```

`player-lens-verify` は既定の Compose プロジェクト名で作られるイメージ名とする。プロジェクト名を変更した場合は `docker compose --profile tools config --images` で対象を確認する。

## 構成と制約

- `cmd/server/`：HTTP 応答、設定、サーバー起動・終了とテストを管理する。
- `compose.yaml`：開発用サーバーと検証用 Go 環境を管理する。
- `Dockerfile`：development と production を分け、Go・実行用のベースイメージを digest で固定する。
- `scripts/`：共通検査と Issue・PR・Release の作成手順を管理する。
- `docs/`：[仕様・ADR・開発手順](docs/README.md) を管理する。

`/healthz` は HTTP サーバーの応答を確認する。外部サービスや調査の成功を保証しない。[HTTP の仕様](docs/specs/http-server.md) に従い、PORT と終了時の期限を扱う。

本番用イメージの検査はソース共有やホストの Go に依存せず、一度ビルドした同じ image ID を使う。これはローカル・PR の動作検証であり、マージ後のリリース対象や Render に配置した成果物の検証とは区別する。イメージの公開と同じ digest の配置、Render のサービス構成は今後導入する。

Go の coverage 閾値・分岐網羅・mutation 検査は未導入とする。GitHub の新しいアプリケーション CI は push 後に対象 SHA の実行結果を確認する。ブランチ保護の制約は [開発手順](docs/development.md#必須チェック設定と残る限界) を参照する。
