# Ready への変更時の重複した規約検査をなくした

[Issue #123](https://github.com/shigaresso/player-lens/issues/123) に対応した。PR #122 統合済みの origin/main（aeda24c）から codex/123-skip-ready-metadata を作成した。

PR 規約検査の ready_for_review トリガーを削除した。コード・本文・ラベルの変更時には検査しているため、Draft から Ready への切り替えだけで同じ内容を再検査する処理をなくした。opened・synchronize・reopened・edited・labeled・unlabeled は維持した。

pr_ci_scope.py も実際の対応イベントに揃え、ready_for_review を直接受け取った場合は未対応として失敗する構成にした。本文・タイトル編集の軽量検査、統合先・不明な編集情報の追加検査、未知・不正入力の拒否は維持した。開発手順と ADR-0002 の現在の方針を更新し、Issue #18 当時の作業記録は保持した。

## Red・Green

最初にワークフローから PR イベントを読み取る回帰テストを追加した。python3 -m unittest discover -s scripts -p test_pr_ci_scope.py は、余分な ready_for_review が含まれることを理由に1件失敗した。ワークフローからそのイベントだけを削除し、同じテストファイルの5件が成功した。

次に scope 判定のテストを対応イベントに揃え、ready_for_review の直接入力を未対応として拒否する期待を追加した。関数が ValueError を返さず、CLI が成功したため2件失敗することを確認した。対応イベントから ready_for_review を除き、同じ5件が成功した。関数・CLI の両方で対象外を拒否し、CLI が成功の出力を作成しないことを確認した。

ワークフローのテストは現在のインラインのイベント配列を読み取るものであり、汎用 YAML パーサーではない。YAML の構文・Actions の設定は既存 actionlint で別に確認する。

## 検査範囲と維持した契約

origin/main と比較し、pr-conventions.yml は ready_for_review の削除以外が同一であることを確認した。他の全ワークフローは変更していない。ジョブ・権限・concurrency・タイムアウトと、アプリケーション検査・main の push・Issue の closed を含むイベントは維持した。

共通の規約検査は必須 pre-commit の bash scripts/check.sh で行い、秘密情報検査・commit-msg と合わせて、実行結果と確定コミットを Issue に記録する。隔離した実フックの検証には python3 scripts/verify_hooks.py を使用する。実 push を行う検証ではなく、アプリ検査と Docker の呼び出し先は検証用の処理へ置き換わる。

Go コード・テスト・依存・Docker の実行構成は変更していないため、今回の変更固有の検証で Go のアプリ検査は再実行していない。coverage 閾値・分岐網羅・mutation（Stryker を含む）は現行では未導入であり、成功した検査として扱わない。

## 参照と残る確認

文書一覧、開発手順の「GitHub での検査」「Issue・PR の作成手順」「各フックの責務」、ADR-0002 の PR 検査の起動方針、ADR-0001 の PR と main 統合後の検証の区別を参照した。

差分のセルフレビュー、文書リンク、端末ユーザー名・ホームパスの非混入を確認した。GitHub 認証と Git の書き込みは許可された権限で実行し、ログはリポジトリ外の一時ディレクトリに置いた。保存先・実データ・依存は追加していない。

push・PR 作成・Issue のクローズ、実 GitHub Actions の起動・非起動確認、人によるレビューは未実施。ローカルの設定・判定の成功を、GitHub 上の実イベント検証済みとは扱わない。
