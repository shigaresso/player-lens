## 背景・目的

PlayerLens の Git・GitHub 運用規約を、ローカルと CI の共通検査へ接続した。参考の sql-drill で見つかった空欄の見逃し、例示の参照、チェック済み完了条件の扱いを改善した。

## 変更内容

- Python 標準ライブラリでコミット・PR・Issue の記載規約を検査した。
- 必須節の実質的な内容、重複、プレースホルダー、参照例、ラベルを判定した。
- Git の新規・削除・複数 ref の更新と GitHub の Issue 実在確認を分離した。
- Lefthook の3フック、Issue フォーム、PR テンプレート、読み取り権限の CI を追加した。
- 固定版ツールのチェックサム確認、準備手順、保護設定の制約、作業記録を追加した。
- Issue・Draft PR の作成前検査を `github.py` に組み込み、正式な作成手順を AGENTS.md と開発手順へ記録した。
- 作成前にラベル・remote のコミット・関連 Issue を確認し、同じ本文を stdin で渡した。check-only と検査中のブランチ更新の拒否を追加した。

## 検証

- `bash scripts/check.sh` が成功した。規約ツール58テスト、空白、Lefthook、actionlint、シェル構文を確認した。
- `python3 scripts/verify_hooks.py` の実フック5テストが成功した。実際の push は行わなかった。
- `python3 scripts/validate.py range origin/main HEAD`、PR 本文案、現在の Issue 本文、文書リンクを確認した。
- 作成 CLI の本文と引数の伝播、失敗時の無書き込みは偽の gh で確認した。実 API では Issue の check-only 成功と未送信 head の PR 拒否を確認し、検証目的の新規 Issue・PR は作成しなかった。
- 状況ごとの Red・Green は `docs/worklogs/3-conventions.md` に記録した。既に成功した状況は回帰テストとして保存した。
- AI によるセルフレビューを実施した。人によるレビューは未実施とした。
- この本文案の作成時点では push・PR 作成を行わず、GitHub 上の CI は未実行とした。PR 作成後に run と対象 SHA を確認して結果を追記する必要がある。
- private リポジトリの保護・Rulesets API が403を返したため必須チェックは未設定とした。公開化・契約変更は行わなかった。

## 参照仕様・ADR

AGENTS.md の言語・Issue・コミット・PR・自動検査、`docs/development.md` の本文検査と各フック、ADR-0001 の導入状態、ADR-0002 の決定と制約を確認した。アプリケーションのデータ・公開デモ要件には変更を加えなかった。

## 関連 Issue

Closes #3
