# Issue #11：最小 HTTP サーバーと Docker Compose

## 対象と状態

[Issue #11](https://github.com/shigaresso/player-lens/issues/11) を最新の origin/main `d2a55c55b92db4a163269d5006cfe13b41d22cbf` から `codex/11-compose-healthz` で実装した。push・PR 作成・Issue のクローズは行わない。GitHub の新しいアプリケーション CI は未実行であり、ローカルの結果を代用しない。

## 参照した契約

AGENTS.md の契約単位の機能開発・Go・テスト・Docker・自動検査、文書一覧、ADR-0001 の決定・配置時の条件・制約、公開デモ仕様の README 要件、開発手順と ADR-0002 のフック・CI の責務を参照した。HTTP の具体的な契約を [仕様](../specs/http-server.md) と Issue に記録した。

## Red・Green とコミット

- PORT：`TestListenAddress` が `listenAddress` 未実装で Red となり、実装後に境界値と不正値を含めて Green になった。
- HTTP：`TestHealthHandler` が `newHandler` 未実装で Red となり、GET・HEAD・404・405 が実装後に Green になった。書き込み失敗の記録は `TestHealthWriteFailure` の未記録による Red、記録追加後の Green を確認した。
- 終了処理：実 HTTP と処理中の終了待ちを確認する結合テストが `serve` 未実装で Red となり、実装後に Green になった。終了期限超過・接続解放・Serve 失敗の原因保持は回帰テストで確認した。
- 起動失敗：`run` 未実装で Red となり、実装後に不正 PORT・使用中ポートの拒否が Green になった。
- push 前検査：アプリケーション検査の失敗を拒否する実フックテストが、呼び出し未接続で期待した終了コード1を返さず Red になった。共通検査を接続し、隔離したテスト用実装で失敗伝播を確認して Green になった。
- 分割前の `56120ad`：Go サーバー、テスト、仕様、固定版 Go の開発環境を追加した。Red と実装を別の共有コミットにせず、意味のある単位にまとめた。

## 履歴の再整理

ユーザーの依頼に従い、未 push の2コミットを次の5コミットへ分け直した。元の先端 `ea46285f2a743c40299a3699122a53814540c5ea` はローカルの `codex/11-compose-healthz-before-split` に保存した。

| コミット | 契約と対応する検証 |
| --- | --- |
| `9b3b337` | Go・Compose の検証基盤と PORT。境界値を含むユニットテスト・整形・vet |
| `1822d5d` | healthz と書き込み失敗。handler のテスト・整形・vet |
| `a65f9dc` | 起動・終了処理と開発用 Compose。ユニット・結合テストを含む race 検査 |
| `47bb08b` | 本番用イメージ。HTTP・PORT・不正設定・SIGTERM/SIGINT の実検査 |
| 「CI とフックのアプリケーション検査を接続した」 | 共通コマンド・CI・フックと関連文書。必須の規約検査と actionlint |

各段階のファイルだけを復元して関連検査を通し、テストと実装を同じコミットに含めた。全コミットで pre-commit と commit-msg を有効にした。既に実装済みの変更を整理する作業なので、新たな Red は作らなかった。元の Red・Green の実記録は上記に残し、分割後の履歴だけで実際の開発順序を証明したとは扱わない。

アプリケーション・設定・検査コードの最終内容は分割前と同じとし、この作業記録だけを履歴の再整理に合わせて更新した。既存の実フック6テストと Watch の実行結果は、同じコードに対する確認結果として再利用した。

## 検証結果

- Compose 内の `go vet ./...` と `go test -tags=integration -count=1 ./...` が成功した。
- `bash scripts/check-app.sh` の整形・静的検査・通常テスト、結合テストを含む race 検査、本番用イメージ検査が成功した。
- 本番用の `sha256:74fc2ec8fcaeee2f910e45035718303e606aa16a6b3ab924a45781c29665e3d5` を一度ビルドし、同じ image ID で8080の HTTP 応答と SIGTERM、10000の HTTP 応答と SIGINT、不正 PORT の非0終了を確認した。実行は非 root・読み取り専用・ソース共有なしで行った。
- 開発用 Compose を PORT=8080、ローカル公開18080で起動し、`/healthz` の応答と正常終了を確認した。
- Compose Watch で応答本文を一時的に変更し、同期・再起動後の実 HTTP 応答を確認した。元のソースへ戻し、再び正常な応答を確認して停止した。
- `python3 scripts/verify_hooks.py` の実フック6テストが成功した。アプリケーション検査の接続と失敗伝播は隔離したテスト用実装で確認した。
- 一時的な未整形 Go ファイルを追加し、共通の unit 検査が非0で拒否することと自動修正しないことを確認した。一時ファイルは削除した。
- 実際の `.git/hooks/pre-push` に現在のブランチの更新情報を stdin で渡し、規約の115テスト・静的検査、Go と race 検査、同じ本番用 image ID のスモーク検査が成功した。remote への push は実行しなかった。
- セルフレビューで HTTP の契約、リクエストを即時キャンセルしない終了処理、listener・goroutine の回収、image ID の再利用、フックからの失敗伝播、未導入事項の記述を確認した。人によるレビューは未実施とする。
- 分割前は本番用イメージ・共通検査・CI・文書を2つ目のローカルコミットへまとめていた。再整理では上記の契約単位へ分け、既存の運用規約検査と実フックの検証を維持した。

## 保証の範囲と残る事項

検証したのはローカル arm64 の Go・HTTP・イメージであり、新 CI の amd64 実行、リリース対象の registry digest、Render 配置、本番 URL の検証は未実施とする。DB・Steam・LLM・MCP・UI・認証、coverage 閾値・分岐・mutation は未導入とし、healthz の成功でそれらを保証しない。ブランチ保護は現行契約の制約により未設定とする。
