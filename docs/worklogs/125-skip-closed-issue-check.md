# Issue クローズ時の重複した規約検査をなくした

[Issue #125](https://github.com/shigaresso/player-lens/issues/125) に対応した。PR #124 統合済みの origin/main（4a8e090）から codex/125-skip-closed-issue-check を作成した。

.github/workflows/issues.yml の closed トリガーだけを削除し、opened・edited・reopened・labeled・unlabeled の5イベントを維持した。クローズ時の検査は内容変更時と同じ記載形式の確認であり、実装の完了判定やマージ・クローズの阻止ではないため、同じ内容の再検査をなくした。

開発手順と ADR-0002 に削除理由と新しい起動条件を記録した。完了条件の確認・チェックは引き続き作業者が行う。PR の opened・labeled と concurrency、main 統合後の規約・アプリケーション検査には変更を加えていない。過去の作業記録は当時の履歴として保持した。

## Red・Green

scripts/test_issue_workflow.py に、ワークフローから issues の起動イベントを読み取る回帰テストを先に追加した。python3 -m unittest discover -s scripts -p test_issue_workflow.py は、余分な closed が含まれることを理由に1件失敗した。

closed の削除後、同じコマンドの1件が成功した。ほかの5イベントを維持したことも同時に確認する。新しいテストは現在のインラインのイベント配列を対象とし、YAML 全体の構文は既存 actionlint で別に検査する。汎用パーサーや新しい依存は導入していない。

## 変更範囲と検証

origin/main と比較し、issues.yml は closed の削除以外が同一であることを確認した。ジョブ・権限・concurrency・タイムアウト、他の全ワークフロー、Go コード・テスト・依存、既存の規約検査処理は変更していない。

必須 pre-commit でステージ済みの秘密情報検査と bash scripts/check.sh の共通規約・Python・actionlint・Lefthook 設定・シェル構文・差分検査を実行し、commit-msg と合わせて結果・確定コミットを Issue に記録する。隔離した実フックの検証には python3 scripts/verify_hooks.py を使用する。実際の push は行わず、Docker・アプリ検査の呼び出し先は検証用の処理へ置き換わる。

Go と Docker の実行構成を変更していないため、今回の変更固有の検証では Go のアプリ検査を再実行していない。coverage 閾値・分岐網羅・mutation（Stryker を含む）は現行では未導入であり、今回の成功した検査に含めない。

## 参照と残る確認

文書一覧、開発手順の「GitHub での検査」「各フックの責務」「Issue・PR の作成手順」、ADR-0002 のイベントと検査の責務、ADR-0001 の PR と main 統合後の検証の区別を参照した。

差分のセルフレビュー、文書リンク・端末ユーザー名とホームパスの非混入を確認した。GitHub 認証と Git の書き込みは許可された権限で実行し、ログはリポジトリ外の一時ディレクトリへ置いた。保存先・実データ・依存は追加していない。

push・PR 作成・Issue のクローズ、GitHub 上の実イベントによる起動・非起動確認、人によるレビューは未実施。ローカルの設定検証を、GitHub Actions の実行確認済みとは扱わない。
