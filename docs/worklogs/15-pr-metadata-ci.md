# Issue #15：PR メタデータ更新時の規約 CI

## 対象と契約

[Issue #15](https://github.com/shigaresso/player-lens/issues/15) を origin/main `d3b4527` から `codex/15-pr-metadata-ci` で進めた。AGENTS.md、文書一覧、開発手順の GitHub 検査・必須チェック、採用済み ADR-0002 の共通検査・読み取り権限・Release の参照先を読んだ。ADR-0001・0003 の既存の実行環境と Go 検査を維持する。

PR 本文・タイトル・ラベルの変更はメタデータ検査だけを行い、コード検査と concurrency・チェック名を分ける。現在の PR と全コミット・関連 Issue のバリデーターは維持する。統合先変更や不明な編集情報は全検査に戻す。

## Red・Green

- 本文・タイトルだけの編集と統合先・不明な編集の分類は、`pr_ci_scope` 未実装で Red、実装後に Green になった。同じ分類規則の入力は名前付きテーブルでまとめた。
- イベントファイルを読み `GITHUB_OUTPUT` へ追記する契約と、不正な JSON・未知のイベントの拒否は CLI 未実装で Red、実装後に Green になった。出力を上書きせず、失敗時は成功した分類を出力しないことを確認した。

## 検証と状態

`python3 -m unittest discover -s scripts -p test_pr_ci_scope.py` の4テストが成功した。フック・CI の設定と共通検査、実 CI の結果は対応後に記録する。

`.tools/actionlint -shellcheck=''` と空白検査が成功した。conventions.yml のソースイベント・main の verify を維持し、PR 検査を pr-conventions.yml の metadata へ移した。編集だけの場合のコード検査の起動・キャンセル・チェック上書きを避けるため、ワークフローと concurrency を分けた。統合先などの編集は PR 側で規約の全検査へ戻す。バリデーターと Release の処理、ローカルフックと Go・本番用イメージの検査は変更していない。AI が差分をセルフレビューした。人のレビューは未実施とする。

変更確定の共通検査は commit フックで、全アプリケーション検査は push フックで実行する。既存フックの8テストは PR のコード検査で確認する。実 CI の対象 SHA と各イベントの結果は Issue・PR へ記録する。PR を一時的に不正な形式へ編集して軽量検査の拒否を確認した後、適切な本文へ戻す。
