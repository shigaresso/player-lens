# Issue #13：Go のリンターと整形・静的検査

## 対象と契約

[Issue #13](https://github.com/shigaresso/player-lens/issues/13) を origin/main `c2f4b8d` から `codex/13-go-lint-format` で進める。AGENTS.md、文書一覧、開発手順のフック・Go 検査、採用済み ADR-0001・0002、公開デモ仕様の README 要件を参照した。coverage・mutation は未導入のままとし、品質目標は変更しない。

## Red・Green の記録

- index の Go ソースを一時コピーして検査する契約は `staged_format` 未実装で Red、実装後に Green になった。空白付きパスと作業ツリーとの差を確認し、index・作業ツリーを保持することを確認した。
- Docker の gofmt が未整形ファイルを返した場合の拒否は `format_snapshot` 未実装で Red、実装後に Green になった。
- Go 以外と削除だけの場合の省略、一時コピーの解放、ビルド出力と整形差分の区別、Docker 失敗の伝播は実装済みの契約として回帰テストで Green を確認した。
- 全体の整形検査と明示的な修正は、`format`・`format-fix` が未対応で Red、対応後に Green になった。検査は書き込まず、修正だけに読み書き可能な mount を渡し、ツールの失敗を保持する。macOS の一時ディレクトリのシンボリックリンクを解決するテスト記述へ修正した。

## 検証結果と状態

`python3 -m unittest discover -s scripts -p test_staged_format.py` の6テストが成功した。Docker による実際の整形検査、リンター、フック・CI の検証結果は確認後に追記する。

- golangci-lint v2.14.0 の公式イメージの index digest と、実バイナリが Go 1.27.0 でビルドされたことを確認した。Go 1.27.1 のソースを実際に解析し、既存の結合テストにある5箇所の errcheck 違反で Red、エラー確認を修正して0件の Green を確認した。`bash scripts/check-go.sh integration` の race を含むテストも成功した。
- `python3 scripts/verify_go_tools.py` で正常・4種類のリンター違反・integration タグ、空白付きパスの部分ステージと入力の保持、全体の整形検査・明示修正を確認した。最初の実検証では mount の `:rw` 上書きでも読み取り専用が残って修正に失敗した。修正専用サービスを選ぶ回帰テストで Red、サービス導入後に Green を確認し、実 Docker でも修正できた。
- `bash scripts/check-go.sh unit` の整形・vet・通常テストが成功した。整形コマンドの3テストと、既存の integration 実行スクリプトの4テストも成功した。
- 実フックのステージ済み Go の拒否は、フック未接続でコミットされて Red、接続後の結果は以下の最終検証へ記録する。
