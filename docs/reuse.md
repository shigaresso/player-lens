# 他のプロジェクトへ開発基盤を移植する

## v0.0.1 の位置づけ

PlayerLens の開発規約と規約検査を保存した版とする。Go アプリケーション、DB、Docker、MCP、Render の実装は含まない。検査スクリプトは Python 標準ライブラリで動くため、対象アプリケーションが TypeScript などでも運用検査を利用できる。

AGENTS.md とアプリケーション仕様には PlayerLens 固有の要件がある。移植時は対象プロジェクトの判断として規約・仕様・ADR を整合させる。ファイルを置いただけで検査やフック・保護設定が有効になるとは扱わない。

## 引き継ぐファイルと調整箇所

| ファイル | 移植時の対応 |
| --- | --- |
| `AGENTS.md` | 日本語、Issue 起点、コミット・PR・Release、参照義務、検証の手順を引き継ぎ、目的・言語・テスト方針・アプリケーション固有の条項を見直す |
| `scripts/` | 作成 CLI、純粋な本文検査、Git/GitHub 検査、テスト、ツール配置・共通検査・実フック検証を一緒に移す |
| `lefthook.yml`、`.lefthook/` | 既存フックとの競合を確認して統合する。scripts と同じパスを維持し、実フックを検証する |
| `.github/ISSUE_TEMPLATE/`、PR テンプレート | 本文の必須節とバリデーターの定義を一致させる。フォームを置くだけで作成前検査が動くとは扱わない |
| `.github/workflows/` | 既存 CI と統合する。Release が要求する `conventions.yml` と `verify` の名前を変更するなら `release.py`・対応テスト・文書も同じ Issue で更新する |
| `.gitignore` | `.tools/` と Python のキャッシュなどの除外を既存設定へ追加する。既存の除外を丸ごと置き換えない |
| `docs/development.md`、`docs/releases.md`、ADR-0002 | 導入状態・OS 対応・GitHub の保護設定・作成手順を対象プロジェクトで検証して記録する |
| `docs/specs/`、ADR-0001、文書一覧 | 保存・公開デモ・Render などの仕様をそのまま流用せず、対象の要件と採用判断へ書き換え、AGENTS.md の参照を整合させる |

`.git/`、`.tools/`、過去の Issue 番号・作業ブランチ・作業記録を新規プロジェクトの履歴として移さない。アーカイブの内容を展開して選び、移植先の既存変更を保護して小さな PR で導入する。

## 導入と確認

1. 移植先の Issue を作成し、現在の origin/main を起点にブランチを作る。本文の必須節、ラベル、既存の Git フック・CI と統合する範囲を決める。
2. リポジトリ名・URL・プロジェクト名・文書参照・言語固有の規約・Issue の参照番号を対象に合わせる。コード内のエラーメッセージや偽 API のテストデータと、実際の設定・コマンド例を区別する。
3. Python 3.12 以降、Git、curl、tar、install と認証済み gh を用意する。現行のツール配置は macOS arm64 / Linux x86_64 を扱い、Lefthook 2.1.9、actionlint 1.7.12 を版・SHA-256 で固定する。対象外の環境は対応を検討してから導入する。
4. 既存フックを確認して `bash scripts/install-tools.sh`、`.tools/lefthook install` を実行する。`.tools/` は各環境で生成し、コミットしない。
5. `bash scripts/check.sh`、`python3 scripts/verify_hooks.py` を実行し、実際の Issue・Draft PR の作成とその CI も移植先で確認する。CLI の `--repo` を移植先の OWNER/REPO にする。
6. 利用可能なブランチ保護・Rulesets に必須チェックを設定し、設定を確認する。契約や権限で利用できない場合は未設定と記録する。
7. Go または TypeScript の実装に必要な検査を共通コマンド・CI へ追加する。規約ツールの成功をアプリケーションのテストの代用にしない。

Release の対象ブランチは現在 `main` に固定している。別の運用なら、Git 範囲検査・CI トリガー・Release の検査・規約・テストを一緒に変更する。CI の必須ジョブ名や対象の変更を、失敗を回避するためだけに行わない。

日本語文字や Markdown の形式は検査できるが、説明と実装の一致・過去形の意味・Issue の完了・ラベルの適切さはレビューで確認する。詳細は [開発手順](development.md) と [Release 手順](releases.md) を参照する。
