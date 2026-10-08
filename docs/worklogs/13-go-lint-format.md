# Issue #13：Go のリンターと整形・静的検査

## 対象と契約

[Issue #13](https://github.com/shigaresso/player-lens/issues/13) を origin/main `c2f4b8d` から `codex/13-go-lint-format` で進める。AGENTS.md、文書一覧、開発手順のフック・Go 検査、採用済み ADR-0001・0002、公開デモ仕様の README 要件を参照した。coverage・mutation は未導入のままとし、品質目標は変更しない。

## Red・Green の記録

- index の Go ソースを一時コピーして検査する契約は `staged_format` 未実装で Red、実装後に Green になった。空白付きパスと作業ツリーとの差を確認し、index・作業ツリーを保持することを確認した。
- Docker の gofmt が未整形ファイルを返した場合の拒否は `format_snapshot` 未実装で Red、実装後に Green になった。
- Go 以外と削除だけの場合の省略、一時コピーの解放、ビルド出力と整形差分の区別、Docker 失敗の伝播は実装済みの契約として回帰テストで Green を確認した。

## 検証結果と状態

`python3 -m unittest discover -s scripts -p test_staged_format.py` の6テストが成功した。Docker による実際の整形検査、リンター、フック・CI の検証結果は確認後に追記する。
