# ADR-0002：規約検査を Python 標準ライブラリで実装する

## 状態

採用。[Issue #3](https://github.com/shigaresso/player-lens/issues/3) でローカルの規約検査・フックと CI の設定を導入した。PR #4 の規約 CI と Issue #5 の Issue イベント CI を GitHub 上で確認した。[Issue #5](https://github.com/shigaresso/player-lens/issues/5) で Release の事前検査を同じ環境へ追加した。必須チェック設定は未設定とする。対象 SHA と結果は [Issue #3](../worklogs/3-conventions.md) と [Issue #5](../worklogs/5-release.md) の作業記録に残す。

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
- Issue・PR の作成は `scripts/github.py` で事前検査と gh の書き込みを一つの手順にする。検査済み本文を stdin で渡し、PR の head/base と Draft を明示する。`--check-only` は読み取りだけとし、自動 push・fork・作成の再試行を行わない。CI の読み取り権限は変更しない。
- Release は `scripts/release.py` で Draft 作成と公開のそれぞれに事前検査を組み込む。本文検査と gh の I/O を再利用し、対象 SHA の remote main と規約 CI・必須ジョブを要求する。保存済み Draft の再確認と公開後のタグ・本文照合を行い、通信失敗では自動再試行しない。CI に書き込み権限を追加して自動公開する方式は採用しない。
- ローカルと CI は同じスクリプト・検査関数を使う。ローカルの形式検査と CI の Issue 実在確認を区別する。
- Issue #143 で一時 Git リポジトリを使う検査から継承した `GIT_*` と system・global 設定を分離する。cwd の変更だけではフックの Git 環境が実リポジトリを参照するためである。スコープ終了時に元の環境を復元し、実コミット・送信対象の検査には適用しない。使い捨ての親リポジトリと linked worktree の実フックで保持を検証する。環境を一時変更する検査の同一プロセス内での並列実行には対応しない。
- Lefthook 2.1.9、actionlint 1.7.12 の公式リリースを版・SHA-256 で固定し、プロジェクト内の `.tools` へ配置する。更新時は版・チェックサム・動作を同じ Issue で確認する。
- `pre-push` は stdin を受け取る script 方式で接続する。コマンド方式の暗黙のファイル差分判定によって、空コミットのメッセージ検査が省略されることを避ける。
- CI の Action はコミット SHA で固定し、既存検査の権限は読み取りに限定する。[ADR-0010](0010-render-deployment.md) の main から手動実行するイメージ公開ジョブだけは packages:write を許可する。Release 手順の権限は変更しない。
- [Issue #15](https://github.com/shigaresso/player-lens/issues/15) で PR の現在の本文・ラベル・全コミット・関連 Issue の確認を `PR 規約検査 / metadata` へ分ける。本文・タイトルだけの編集とラベル変更は共通テスト・実フック検証を再実行しない。統合先変更や不明な編集は規約の全検査へ戻す。コード検査の `規約検査 / verify` と concurrency・ジョブ名を分け、メタデータ更新によるキャンセルや成功の上書きを避ける。条件で同じ verify ジョブのテストだけを省略する方式は採用しない。
- [Issue #18](https://github.com/shigaresso/player-lens/issues/18) で、Draft PR 中に全検査の成功を確認してからレビュー準備へ進む運用に合わせ、`ready_for_review` はメタデータ検査だけとした。[Issue #123](https://github.com/shigaresso/player-lens/issues/123) では、この切り替えを PR 規約検査の起動条件からも外す方針へ変更した。コード・本文・ラベルの変更時に検査しており、切り替えだけで同じ内容を再検査する利点が小さいためである。PR の作成・コミット追加・再開・編集・ラベル変更時のメタデータ検査と、Draft PR を含む作成・コミット追加・再開、main 更新・手動実行のコード検査は維持する。レビュー準備へ進む前に現在のコード・メタデータに対応する検査結果を確認し、統合後の SHA は main で検査する。
- [Issue #9](https://github.com/shigaresso/player-lens/issues/9) で通常マージを基本とし、main 更新専用の `main-range` を追加する。作業コミットと PR・ローカルフックの検査は維持し、親が2つのコミットと同一リポジトリの main 向けマージ済み PR の統合 SHA・head を照合できた場合だけ、自動生成の統合メッセージを免除する。単にマージコミットを一律除外する方式は、未確認の統合を免除するため採用しない。
- [Issue #125](https://github.com/shigaresso/player-lens/issues/125) で Issue 規約検査の closed トリガーを削除する。クローズ時の検査を残す案と比較し、内容変更時と同じ形式の再検査をなくす方針を採用した。クローズ後の形式検査は完了判定やマージ・クローズの阻止にはならないためである。opened・edited・reopened・labeled・unlabeled と従来の検査内容を維持し、完了条件の確認・チェックは作業手順に従う。main 統合後の規約・アプリケーション検査と PR メタデータ検査は変更しない。

## 秘密情報の自動検査（Issue #97）

検出エンジンの自作、GitHub の機能だけへの依存、既存 CLI の利用を比較し、公開前点検で使用した Gitleaks v8.30.1 の CLI を採用する。標準の検出ルールを使い、Python は版確認・結果の秘匿・失敗伝播と既存の Git 更新検査への接続を担当する。独自 Action や実行用の Go 依存を増やさず、公式バイナリと SHA-256 を既存 install-tools.sh で固定する。Gitleaks 公式は機能追加を終了しセキュリティ修正を継続する方針を示しており、更新時に保守状況とルール変更を確認する。

- pre-commit はステージ済み差分、commit-msg はメッセージ、pre-push は既存の規約検査が求める全送信コミットを検査する。履歴にはマージ解決の差分とコミットメッセージを含める。
- conventions.yml に独立した secrets ジョブを追加し、同じ CLI で取得済み履歴を検査して、実 Gitleaks の正常・拒否・部分ステージ・資源保持等も検証する。CI は送信後の検査であり、フックや権限制御の代用とはしない。
- 標準 allowlist は引き継ぐ。go.mod/go.sum・画像・依存物などのパス除外、既知の値・環境変数参照の許可があり、全ファイル・任意のパスワード・端末情報を網羅する保証とはしない。独自の広い除外・baseline は設けない。
- ツール不在・版違い・実行失敗・結果破損も非0終了とする。プロジェクトの設定を明示し、GITLEAKS_* 環境変数や gitleaks:allow、独自 ignore ファイルで検査条件を変えない。検出値を出力・artifact に残さず、一時結果を削除する。誤検知の例外は理由・範囲・検証を記録して判断する。
- Issue・PR 本文、添付、Actions ログ、配布物の継続巡回や GitHub Secret scanning/Push protection の設定は別範囲とする。運用規約の内容確認は維持する。

実装・ローカル検証と未実施の GitHub CI は [Issue #97 の記録](../worklogs/97-secret-scanning.md) で区別する。設定と手動コマンドは [開発手順](../development.md#秘密情報の自動検査) に記載する。

## 利点と制約

- 規約検査を Go アプリケーション・DB・Docker の導入より先に追加できた。規約ツールには Python、Git と固定版の検査ツールが必要となる。Issue #11 で Go の検査を別スクリプトとして導入し、push 前と独立 CI ジョブへ接続した。
- 現在のツール配置スクリプトは macOS arm64 と Linux x86_64 を扱う。別 OS・アーキテクチャの対応は別 Issue で追加する。
- 見出しや日本語文字の存在は機械的に確認できるが、日本語の自然さ、過去形、実施内容の真偽、仕様を読んだこと、Issue の完了はレビューで判断する。
- 作成 CLI の利用を開発規約へ記録する。ブラウザや直接の gh による作成を遮断するものではなく、作成後の CI を残す。事前検査と書き込みの間の remote 更新を原子的に防げるとは扱わない。
- 手動でフックを回避することは規約で禁止する。機械的な強制には GitHub の必須チェックも必要だが、現時点の private リポジトリでは保護・Rulesets API が 403 を返した。公開化や契約変更はこの Issue で行わない。
- Go の検証は Issue #11 で追加した Compose の検証環境で行う。Python の運用ツールをホスト・CI で実行することは、Go アプリケーションの開発・本番環境の一致を保証しない。隔離したフックの検証は Docker 検査の接続と失敗伝播をテスト用実装で確認し、実資源の検査とは区別する。
- 通常マージの例外確認には GitHub の読み取り接続が必要となる。確認できない候補は従来の形式検査を受け、通信失敗を成功扱いにしない。GitHub 上の PR 記録との対応を確認するものであり、統合の生成者を署名で証明するものではない。Squash メッセージの免除や既存履歴の書き換えは行わない。

## 参照

- [ADR-0003：Go の整形・リンターとフックへの接続](0003-go-format-and-lint.md)
- [開発手順と検査の契約](../development.md)
- [Release 手順と制約](../releases.md)
- [ADR-0001](0001-runtime-and-deployment.md)
- [Lefthook の stdin](https://lefthook.dev/usage/features/pass-stdin/)
- [固定版の script 実行経路](https://github.com/evilmartians/lefthook/blob/v2.1.9/internal/run/controller/command/build_script.go)

- [Gitleaks v8.30.1 の公式仕様](https://github.com/gitleaks/gitleaks/tree/v8.30.1)、[標準設定](https://github.com/gitleaks/gitleaks/blob/v8.30.1/config/gitleaks.toml)、[配布物](https://github.com/gitleaks/gitleaks/releases/tag/v8.30.1)
