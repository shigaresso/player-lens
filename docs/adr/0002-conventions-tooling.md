# ADR-0002：規約検査を Python 標準ライブラリで実装する

## 状態

採用。[Issue #3](https://github.com/shigaresso/player-lens/issues/3) でローカルの規約検査・フックと CI の設定を導入する。GitHub 上の CI 実行と必須チェック設定は未確認・未設定とする。

## 背景と選択肢

アプリケーション実装の前に Git・GitHub の運用規約を検査する。参考の sql-drill には Node のバリデーターがあるが、空欄や例示の判定、チェック済み完了条件、関連番号の実在確認を改善する必要がある。

| 選択肢 | 判断 |
| --- | --- |
| 参考の JavaScript を移植する | Node の実行環境と元の判定を引き継ぐことになるため、採用しない |
| Go の検査用 CLI を先に実装する | Go モジュール・検証環境を先に導入する必要があり、アプリケーションの初期構成とは別に判断するため、今回採用しない |
| Python 標準ライブラリで実装する | この作業環境に Python 3.12 があり、外部パッケージなしで文字列、JSON、Git/gh、ユニットテストを扱えるため、採用する |

## 決定

- 規約ツールは Python 3.12 以降の標準ライブラリで実装する。Go アプリケーションの言語・パッケージ規約は変更しない。
- 本文を判断する純粋な処理と、ファイル・Git・GitHub を扱う CLI を分離する。通常のテストは実 GitHub に接続しない。
- GitHub の読み取りは `gh api --method GET` へ委譲し、認証情報を Git・イメージ・引数へ埋め込まない。通信失敗を成功にしない。
- ローカルと CI は同じスクリプト・検査関数を使う。ローカルの形式検査と CI の Issue 実在確認を区別する。
- Lefthook 2.1.9、actionlint 1.7.12 の公式リリースを版・SHA-256 で固定し、プロジェクト内の `.tools` へ配置する。更新時は版・チェックサム・動作を同じ Issue で確認する。
- `pre-push` は stdin を受け取る script 方式で接続する。コマンド方式の暗黙のファイル差分判定によって、空コミットのメッセージ検査が省略されることを避ける。
- CI の Action はコミット SHA で固定し、権限は読み取りに限定する。

## 利点と制約

- Go アプリケーション・DB・Docker の導入を先回りせず、規約検査を追加できる。開発者には Python、Git と固定版の検査ツールが必要となる。
- 現在のツール配置スクリプトは macOS arm64 と Linux x86_64 を扱う。別 OS・アーキテクチャの対応は別 Issue で追加する。
- 見出しや日本語文字の存在は機械的に確認できるが、日本語の自然さ、過去形、実施内容の真偽、仕様を読んだこと、Issue の完了はレビューで判断する。
- 手動でフックを回避することは規約で禁止する。機械的な強制には GitHub の必須チェックも必要だが、現時点の private リポジトリでは保護・Rulesets API が 403 を返した。公開化や契約変更はこの Issue で行わない。
- Go の検証は今後 Compose の検証環境で行う。Python の運用ツールをホスト・CI で実行することは、Go アプリケーションの開発・本番環境の一致を保証しない。

## 参照

- [開発手順と検査の契約](../development.md)
- [ADR-0001](0001-runtime-and-deployment.md)
- [Lefthook の stdin](https://lefthook.dev/usage/features/pass-stdin/)
- [固定版の script 実行経路](https://github.com/evilmartians/lefthook/blob/v2.1.9/internal/run/controller/command/build_script.go)
