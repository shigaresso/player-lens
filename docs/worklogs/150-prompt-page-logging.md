# Issue #150：プロンプト作成画面の応答ログ

## 対象と判断

[Issue #150](https://github.com/shigaresso/player-lens/issues/150) に対応し、最新 origin/main の `175eb5c3b7cae507f589108ebd0d3a7cdc60001b` から `codex/150-prompt-page-logging` を作成した。

文書一覧、公開デモ仕様「プロンプト作成画面」、HTTP 仕様の生存確認ログ、Controller 一覧、ADR-0013・0017・0021 と開発手順を読み、既存の slog/context と注入の方針を適用した。応答観測とブラウザ操作の境界、ServeContent の結果取得の判断は [ADR-0022](../adr/0022-prompt-page-structured-logging.md)、項目・レベル・失敗は [公開デモ仕様](../specs/public-demo.md#プロンプト作成画面の応答ログissue-150) に記録した。

Controller は応答・計測・ログ接続、page.Serve は標準応答の結果観測、logging.PromptPage は安全な固定属性と分類を担当する。起動側から logger と時計を渡し、生成時には使わない。通常 return 時だけ1回記録し、panic は回復しない。ルート定義とログは PromptPath を共有する。

## 契約と Red / Green

| 契約 | 意図した Red | Green の内容 |
| --- | --- | --- |
| HTTP 応答の観測 | page.Serve が未定義 | GET/HEAD、部分・複数範囲、304・412・416と本文・ヘッダーを維持した |
| 書き込み失敗の観測 | Response.WriteErr が未定義 | 通常・部分・複数範囲・拒否本文の失敗、短い書き込みを保持し、再送しなかった |
| 完了属性と状態分類 | logging.PromptPage が未定義 | 固定属性・時間・1イベント、正常・4xx・5xxの分類を記録した |
| 安全な書き込み失敗 | PromptPage.WriteErr が未定義 | キャンセル・期限・切断・通信タイムアウト・未知の分類と失敗優先、原因の非漏出を確認した |
| Controller との接続 | logger・時計を渡す生成 API が未実装 | 応答・時間・非実行・1イベント・入力非漏出を確認した |
| 本番ログの検査 | check_prompt_page_logs が未定義 | GET/HEAD がそれぞれ1件で、安全な固定項目・値だけを持つことを検査した |

未使用 import の記述ミスを修正してから未実装 API だけの Red を再確認した。ログの結果判定を属性生成の前へ整理した際も、テストを変えず Green を維持した。context の保持・ログ出力先の失敗・panic・実プロセスの結合は、既存実装で Green を確認した回帰テストであり、形式のための失敗を作っていない。

実装と単体契約は `491e0bc`、実プロセスと本番イメージの検査は `5c3eba9` に分けた。

## 検証結果（2026-11-02）

- 固定版 Go の関連テストで、応答・ログ属性・原因分類・context・出力先失敗・panic の契約を確認した。通常テストのログ出力先は破棄先かメモリで、ターミナルへ出力しない。
- `go test -tags=integration -race -count=1 ./cmd/server -run 'TestPromptPageLogOnStderr|TestPromptBuilderWithHTTP'` 相当の Compose 検証が成功した。実バイナリの起動・stderr・HTTP200/206/304/416、404/405の非記録、healthz の別イベント、stdout の非出力と正常終了を確認した。
- `bash scripts/check-app.sh` が成功した。JavaScript 47ケース、Go の整形・vet・通常テスト、レイヤー依存・Controller 配置、通常／integration の golangci-lint、全 integration を含む race と本番イメージを確認した。初回の staticcheck が指摘したテストの条件分岐は switch へ整理し、最終検査で解消した。
- 同じ本番 image ID `sha256:24af4a9d0542090193e5483dd81b7ba8b10835e511e8081eb542943f75309e0c` で、GET/HEAD の画面・ログ、既存 HTTP/MCP、公開設定の拒否、PORT・制限の不正設定、SIGINT/SIGTERM を検証した。
- 必須 pre-commit／commit-msg で Python 198テスト、差分・Lefthook・Actions・シェル構文、ステージ済み Go 整形、規約・秘密情報の検査が成功した。端末固有ホームパスと未追跡・ステージ済みの内容も確認した。

実行ログはリポジトリ外、実プロセスのバイナリはテスト一時領域へ置いた。新たな永続保存先・実データ・認証情報・本番 goroutine・buffer は追加せず、ignore の変更は不要だった。ブラウザ内の処理と画面資産は変更していないため、画面操作の手動反復は行っていない。

新しい GitHub CI、Render 配置、実 Steam・LLM の呼び出しは実施していない。coverage 閾値・分岐・mutation は未導入のままで、検査成功を100%の達成とは扱っていない。push・PR 作成・Issue クローズは行わず、ローカルコミットで停止する。
