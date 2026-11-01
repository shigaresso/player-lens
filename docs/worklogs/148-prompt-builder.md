# Issue #148：MCP 分析依頼の作成画面

## 対象と判断

[Issue #148](https://github.com/shigaresso/player-lens/issues/148) に対応し、origin/main の `8a6354c` から `codex/148-prompt-builder` を作成した。トップページで依頼文を作り、収集・分析の実行は登録済み MCP を使うクライアント AI に任せる。

[公開デモ仕様](../specs/public-demo.md#プロンプト作成画面issue-148)、[HTTP 仕様](../specs/http-server.md#起動時の生成と経路登録issue-74)、[MCP の入力仕様](../specs/review-mcp.md#接続と入力)、[Controller 一覧](../handler-routing.md) と ADR-0001・0002・0003・0011・0013 を参照した。入力上限は MCP schema と共有し、日本語という画面の初期選択は MCP の省略時の英語と区別した。純粋な画面内変換に、新しい application/domain は作らなかった。

## 契約ごとの Red / Green

| 契約 | Red の確認 | Green とコミット |
| --- | --- | --- |
| schema の既定値 | language などの default がない | 省略時と明示した既定値が一致。`d3ead9b` |
| 商品 URL の解析 | JS module が未実装 | URL・ID の正常／拒否と境界値。`6d54614` |
| 条件から依頼文を生成 | generatePrompt が未実装 | 6引数・分析指示・条件の拒否。`379a86d` |
| コピー成功／失敗 | copyPrompt が未実装 | 成功・権限拒否・API 不在・空文字。`379a86d` |
| HTTP 画面と登録 | 生成 API 未定義、登録 API の引数不一致 | GET/HEAD と schema・CSP、POST405・子パス404。`2fffc62` |
| 画面の英語選択 | MCP 既定値を日本語へ変えると英語を拒否 | 画面の選択肢を独立。`5a30b7d` |
| 本番の画面検査 | check_prompt_builder が未実装 | 画面・形式・HEAD の検査。`0b4d833` |
| 配置前の追加 CI | frontend-unit の欠落・失敗・未完了を拒否しない | 新ジョブも必須として拒否。`0b4d833` |

実 HTTP の並行 GET/HEAD・405/404・healthz と、入力定義の HTML エスケープ・変換失敗の追加テストは、既存実装で Green を確認した回帰テストである。形式のための失敗は作っていない。

## 最終検証（2026-11-02）

- `bash scripts/check-app.sh`：固定版 Node.js の47ケース、Go の整形・vet・通常テスト、golangci-lint（通常／integration）、race 付きの全 integration テスト、本番イメージが成功した。
- 同じ本番 image ID `sha256:9bd85d4b86622af02c9f66f12dcb5ce9919e428200ce56b192d5f9148f480a18` で root の GET/HEAD、POST405・未定義404、既存 MCP・公開設定の拒否、PORT・制限の不正設定、SIGINT/SIGTERM を確認した。Node.js は本番に含めない。
- 必須 pre-commit／commit-msg：Python 196テスト、差分、Lefthook・Actions・シェル構文、ステージ済み Go 整形と秘密情報、コミット規約が成功した。
- ブラウザ：初期値、URL 解析、Render／ローカル／独自名、評価・言語・件数、生成・コピー、変更後の無効化、不正 URL と件数・ページ数のエラー、再生成を確認した。390×844でも横はみ出しがなく、確認時のコンソールの警告・エラーはなかった。
- コピーの権限拒否・API 不在は自動テストで確認した。実ブラウザでの権限拒否と JavaScript 無効化は手動確認していない。

画面の生成は実 Steam・LLM を呼ばない。新しい CI の実行、公開配置、実 AI での生成文の遵守は未確認。coverage 閾値・分岐網羅・mutation の導入や達成を示すものではない。

入力・生成文のファイル／DB／ログ保存は追加していない。Node.js は npm 依存を持たず、テストソースは読み取り専用、テスト時のネットワークは無効にした。診断ログと画面確認用の画像はリポジトリ外へ置き、Git と本番イメージへ含めていない。

push・PR 作成・Issue クローズ・Render 配置は行わず、ローカルコミットで停止する。
