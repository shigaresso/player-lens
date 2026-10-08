# Issue #25：レビュー収集 MCP の作業記録

## 対象と方針

最新 origin/main `f516fc831f77b61289e01ba84105ac491692337e` から `codex/25-review-mcp` を作成した。[Issue #25](https://github.com/shigaresso/player-lens/issues/25) の数値上限と失敗契約を実装前に更新した。push・PR 作成・Issue クローズは行わない。

データ・調査仕様の収集条件と部分失敗、収集仕様の入力・走査・期限、HTTP サーバー仕様の期限と終了、公開デモ仕様の状態、開発手順、ADR-0001〜0006 を参照した。追加の仕様と採用理由は review-mcp.md と ADR-0007 に記録した。

## 依存と環境

公式 SDK v1.8.0 を固定し、Docker の Go 1.27.1 で `go get github.com/modelcontextprotocol/go-sdk/mcp@v1.8.0` を実行した。go.mod・go.sum、Docker の依存取得と Watch 対象を追加した。環境・仕様の変更には形式だけの Red を作らない。

## 実装・検証の状態

契約単位の Red・Green、共通検査と実際の結果を完了時に追記する。ChatGPT アプリの手動接続はまだ実施していない。SDK の自動検証とは区別する。
