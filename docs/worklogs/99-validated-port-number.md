# 検証済みポート番号と待ち受けアドレスの構築を分離した

## 背景・目的

[Issue #99](https://github.com/shigaresso/player-lens/issues/99) に対応する。最新の origin/main（b3f839c）から codex/99-validated-port-number を作成した。既存の main と作業履歴を保持し、upstream は設定していない。

## 検証済みポート番号

cmd/server/adapter/port_number.go に ValidatedPortNumber と NewValidatedPortNumber を追加した。生成時に PORT の入力を検査し、String は正規化された10進数の番号だけを返す。内部値は非公開の uint16 とし、変更 API を追加していない。空入力・明示した8080・型のゼロ値は同じ既定値を表す。生成と値の取得は純粋な処理であり、利用時に入力解析や範囲検査を繰り返さない。

### Red と Green

Docker Compose verify で、次の順に関連テストを実行した。各コマンドに -count=1 を指定した。

1. TestNewValidatedPortNumber を先に追加し、型と生成 API の未定義だけでコンパイル失敗する Red を確認した。入力の正常値・上下限・先頭のゼロ、0・範囲外・符号・空白・非数字・全角数字の拒否と、失敗時のゼロ値と PORT の説明を対象にした。
2. 本番コードだけを追加し、TestNewValidatedPortNumber の Green を確認した。
3. TestValidatedPortNumberDefaults を追加し、明示した8080・08080がゼロ値と等しくならない2ケースの Red を確認した。
4. 生成関数で既定値の内部表現を揃え、両テストの Green を確認した。

実行したコマンドは go test -count=1 ./cmd/server/adapter -run に各テスト名を指定したもの。最終確認では '^(TestNewValidatedPortNumber|TestValidatedPortNumberDefaults)$' を指定した。

値型とテストを 2c4587b にコミットした。pre-commit のステージ済み秘密情報検査・Python 182テスト・静的検査・index の Go 整形検査と、commit-msg の秘密情報・規約検査が成功した。

## 起動処理への接続

run は最初に NewValidatedPortNumber のエラーを処理し、待ち受け開始の直前に net.JoinHostPort("", portNumber.String()) でアドレスを組み立てるようにした。全インターフェースの指定と、実際の待ち受けの責務を起動処理に残した。旧 config.go・ListenAddress と旧テストは削除し、入力の契約は新しい生成 API のテストへ引き継いだ。互換ラッパーや新しいパッケージ・依存は追加していない。

切り替え後に Docker Compose verify で go test -count=1 ./cmd/server/... を実行し、通常のサーバー関連テストの成功を確認した。

## 全体検証

bash scripts/check-app.sh が終了コード0で成功した。

- gofmt の差分検査、go vet ./...、go test -count=1 ./... が成功した。
- 固定版 golangci-lint は0 issuesだった。通常と integration タグのコードを検査した。
- Git の失敗・不在・checkout の不在という3条件で、Go の実行と子ビルドが成功した。
- go test -tags=integration -race -count=1 ./... が成功した。これは通常テストと結合テストの両方を含む。既存の待ち受け失敗・起動失敗の構造化ログ・接続解放・終了待機と、ローカルの実 MCP SDK 連携を確認した。
- 本番用 image ID sha256:2bc15a9aa8e5c7b8a8ff5e2eb697268429dbb67af9a4535117faf916713982f3 を一度ビルドして、同じイメージの非 root 設定、ソース共有なしの起動、HTTP/MCP、ローカル／公開 Host・Origin、PORT=8080／10000、不正設定での非0終了、SIGTERM／SIGINT の正常終了を確認した。実 Steam へのリクエストは行っていない。
- 変更文書のローカルリンク89件の参照先が存在することを確認した。アンカーは対象外とした。

### AI のセルフレビュー

入力の検査は生成関数に集約し、private な内部値の構築も同じ関数に置いた。uint16 の内部値は0を既定値として扱い、それ以外は有効な範囲となる。生成済みの値を利用する際の範囲検査や例外的な拒否を追加していない。明示した既定値はゼロ値と同じ表現にし、値としての比較を揃えた。

旧テストの空入力・正常値・上下限・先頭のゼロ・不正値を新テストへ引き継ぎ、PORT のエラー文言と不正設定の早期終了を維持した。net.JoinHostPort は空 Host を指定し、従来と同じ待ち受けアドレスを作る。依存の構築・defer・listener の所有・ログ・Serve の呼び出しは変更していない。人によるレビューは未実施。

## 検証環境

Go の実行は固定版 Docker Compose を使った。最初のコミット前整形検査では既存の Docker 認証ヘルパーが応答待ちとなり、そのビルドを終了したためフックが失敗した。認証情報のない一時 DOCKER_CONFIG と既存の Docker ソケットを指定して同じフックをすべて再実行し、成功した。既存の認証情報やプロジェクトの設定・検査条件は変更していない。一時設定・ログはリポジトリ外に置き、Git・Docker のビルドコンテキストには含めていない。

push・PR 作成・Issue のクローズは行っていない。pre-push・送信済み head/base の PR 作成検査・コード CI は未実施とする。実 Steam／LLM 接続・Render 配置も対象外とした。coverage 閾値・分岐・mutation（Stryker を含む）の検査は現行では未導入であり、品質目標の100%達成を示す結果とは扱わない。

## 参照仕様・ADR

- [HTTP サーバー仕様](../specs/http-server.md) の「設定と終了の契約」「検証との対応」
- [ADR-0013](../adr/0013-domain-folder-layers.md)、[ADR-0015](../adr/0015-collection-options-boundary.md)、[ADR-0018](../adr/0018-validated-server-port.md)
- [開発手順](../development.md) の「Go・Compose・本番用イメージの検査」「各フックの責務」

## 関連 Issue

Closes #99
