# PlayerLens 文書一覧

## 文書の役割と状態

- [AGENTS.md](../AGENTS.md) は開発手順、文書を読む条件、検証と Git・GitHub の規約を定める。
- 仕様書はアプリケーションが満たす要件と、正常・失敗時の期待結果を定める。
- ADR は設計判断の背景、比較、決定、利点と制約を残す。「採用」は方針の決定であり、実装済みを意味しない。
- 文書の要件は実装時の契約とする。未決定事項を実装前に対応 Issue で具体化し、検証結果は Issue・PR に記録する。

## 変更前に読む文書

| 文書 | 対象となる変更 | 状態 |
| --- | --- | --- |
| [データ・調査仕様](specs/data-and-research.md) | 保存、収集、分析、レポート、ジョブと実行上限 | 基本要件を定義した。保存方式・具体的な上限などは未決定、機能は未実装 |
| [公開デモ仕様](specs/public-demo.md) | 公開 UI、認証、利用期限、実行制限、README | 基本要件を定義した。公開範囲・認証方式などは未決定、デモは未配置 |
| [最小 HTTP サーバー仕様](specs/http-server.md) | healthz、PORT、開発用の公開ポート選択、サーバーの期限・終了処理 | Issue #11 で実装した。ローカル検証結果は作業記録へ記載する |
| [ADR-0001：Go・Docker・Render による実行・配置](adr/0001-runtime-and-deployment.md) | 実行環境、開発・検証用 Docker、本番用 CI、Render 配置 | 採用。Compose・本番用イメージと検証 CI の設定を追加した。配置経路は未導入 |
| [開発手順と規約検査](development.md) | 規約バリデーター、Git フック、Issue・PR の作成手順とテンプレート、CI | ローカル検査と PR・Issue イベントの規約 CI を確認した |
| [Release 手順](releases.md) | Draft の作成・公開、本文・タグ・対象・CI の事前検査 | Issue #5 で CLI と通常テストを追加した。実 CI・公開結果は Issue に記録する |
| [移植ガイド](reuse.md) | 他のプロジェクトへの規約・検査・フック・CI の導入 | v0.0.1 の移植対象と調整箇所を記録した |
| [ADR-0002：規約検査の実行環境](adr/0002-conventions-tooling.md) | Python、検査ツール、フック・CI の責務 | 採用。保護設定は現時点の契約では利用できず未設定 |
| [ADR-0003：Go の整形・リンター](adr/0003-go-format-and-lint.md) | Go の整形、リンター、検証用 Docker、フック・CI | 採用。固定版の解析と index の整形検査を導入した。実 CI の結果は Issue・PR に記録する |

変更が複数の範囲にまたがる場合は、対応する文書をすべて読む。参照した節と ADR、要件に対応する検証を Issue・PR に記録する。ADR を追加・置換するときは、この一覧と適用範囲を更新する。

## 現在の導入状況

Issue #11 で Go モジュール、最小 HTTP サーバー、Dockerfile と Compose を追加した。DB・レビュー収集・MCP・公開デモは未導入とする。ローカル検証と新 CI の実行状況は [Issue #11](worklogs/11-http-server.md) に記録する。Issue #3 で Python の規約ツール・Git フック・テンプレート・CI の設定を追加した。PR #4 と Issue #5 の規約 CI を GitHub 上で確認した。Issue #5 で Release CLI と移植ガイドを追加した。保護設定は未設定とし、検査の成功だけでマージの機械的な制限や AI の遵守を保証しない。対象 SHA と結果は [Issue #3](worklogs/3-conventions.md) と [Issue #5](worklogs/5-release.md) の作業記録に残す。

導入が進んだら、対応 PR でこの状態を更新する。`docs/use-cases.md` は最初の操作を導入する際に作成する。自動生成の仕様目次は別 Issue で形式と検査を決定してから導入する。
