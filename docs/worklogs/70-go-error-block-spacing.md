# Go の処理とエラー対応のまとまりを空行で明確にした

## 背景・目的

[Issue #70](https://github.com/shigaresso/player-lens/issues/70) に対応した。処理と直後のエラー対応を1セットとして読み、その後の処理と空行で区別する方針を規約と検査へ反映した。

最新 origin/main（079f50261ab5f67eb8bf2b50d1453854289c7aa9）から codex/70-go-error-block-spacing を作成した。既存ブランチを保持し、upstream は設定していない。push・PR 作成・Issue のクローズは行わず、ローカルで確認できる状態にした。

## 変更内容

- 固定版 golangci-lint の wsl_v5 で default: none から err・after-block・after-defer の3規則を有効にした。既存4リンター、integration タグ、版・digest を維持した。
- 処理と直後のエラー確認を隣接させ、ブロック後と最後の defer 後の次の文を空行で区切った。エラー確認直後の取得変数を参照する defer と、連続する defer は許可した。
- 実装34ファイル・テスト91ファイルに計671行の空行を追加した。文・条件・文字列・コメント・テストの入力や期待値は変更していない。空行適用を 5d35e2b に分けた。
- AGENTS.md・開発手順・ADR-0003 に方針と適用箇所、機械的な検査の限界を記録した。実 Go ツールの入力例と入力保持の確認を追加した。

## 検証

### 対象規則の Red・Green

実 Go ツール検証の入力例を追加し、未設定のため違反を拒否できず 0 issues になる Red を確認してから、対応する設定を順に追加した。既存コードの空行整理には動作変更のための Red を作っていない。

| 契約 | Red | Green |
| --- | --- | --- |
| 実行と直後のエラー確認を隣接させる | err 未設定で、空行による分断を拒否できなかった | err を有効にし、正しいセットは許可し分断は wsl_v5 の違反で拒否した |
| ブロック後の次の処理を区切る | after-block 未設定で、次の文との区切り不足を拒否できなかった | after-block を有効にし、通常と integration の区切り不足を拒否した |
| 最後の defer 後を区切る | after-defer 未設定で、次の文との区切り不足を拒否できなかった | after-defer を有効にし、区切り不足を拒否した。資源取得・エラー確認直後の defer と連続 defer は許可した |

最後の規則の Green 確認中に Docker のコンテナ起動 API が一度500を返した。デーモンの応答を確認して同じ検証を再実行し、成功した。検査や期待結果を緩和していない。

### 全体検証と入力保持

- `python3 scripts/verify_go_tools.py` が成功した。リンターの13入力例（既存6・追加7）、integration タグ、各入力の保持、部分ステージでの index・作業ツリーの保持、全体の gofmt 検査と明示的な修正を確認した。
- 修正時は別パスへソースをコピーし、一時コピーだけに `golangci-lint run --config .golangci.yml --enable-only wsl_v5 --fix ./...` を実行した。全ファイルを比較し、空行を除いた行が同一であることを確認してから125ファイルへ反映した。既存の読み取り専用サービスや Compose 設定は変更していない。
- 基点の cmd・internal の全136 Go ファイルと変更後を Go の scanner で比較し、位置情報を除くトークン種別・リテラル・コメントが全て同一であることを確認した。raw 文字列内の改行もリテラルとして比較した。
- `bash scripts/check-app.sh` が成功した。gofmt・vet・通常テスト、lint（0 issues）、Git 非依存3条件、race 付き通常／結合、本番用イメージを検証した。
- 本番用 image ID `sha256:f39799c850fb1973c8941578b0b4d3c50888ed0997a3f8492b4601d43831dd5c` を一度ビルドし、同じイメージで HTTP/MCP・Host/Origin・PORT・不正設定・SIGTERM/SIGINT を確認した。
- 空行適用のコミットで、必須 pre-commit の Python177テスト・静的検査と index の gofmt、commit-msg が成功した。PR 本文として使える本記録のローカル規約検査も成功した。最終のコミット範囲・差分と Issue 本文の確認結果は Issue に記録する。

### AI のセルフレビューと制約

Go の差分は空行の追加だけで、関数・型・interface・依存、通信・利用枠・context・資源の解放順序、公開形式とテストの期待結果を維持した。新しい対象規則に必要な入力例以外の規則を有効にせず、除外と nolint を追加していない。

pre-commit は index の gofmt、pre-push と go-lint CI は読み取り専用 lint という責務を維持した。gofmt と format-fix 自体は新しい空行を自動生成しない。wsl_v5 は文の形を検査し、処理の意味や全エラー確認の対応までは保証しない。ブロックを持たない文同士の意味上のまとまりはレビューで判断する。人によるレビューは未実施。

GitHub 認証と Docker は権限付きで実行した。既存の一時 Docker 設定を再利用し、認証設定・フック・検査の省略は行っていない。push 前のためコード CI と送信済み head/base による PR 作成検査は未実施で、CI の完了待機も行わない。実 Steam/LLM/ChatGPT・配置、coverage 閾値・分岐・mutation（Stryker を含む）の導入・実行は対象外。

## 参照仕様・ADR

- docs/development.md「各フックの責務」「Go・Compose・本番用イメージの検査」「Go の整形修正とリンターの更新」「Go の空行規則」
- ADR-0002 の共通検査と読み取り専用 CI、ADR-0003 の固定版ツール・入力保持と今回の空行規則。ADR-0001 の既存の Docker 検証環境。
- wsl の公式 CHECKS.md の err・after-block・after-defer、golangci-lint の公式 wsl_v5 設定。

## 関連 Issue

Closes #70
