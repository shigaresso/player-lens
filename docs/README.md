# PlayerLens 文書一覧

## 文書の役割と状態

- [AGENTS.md](../AGENTS.md) は開発手順、文書を読む条件、検証と Git・GitHub の規約を定める。
- 仕様書はアプリケーションが満たす要件と、正常・失敗時の期待結果を定める。
- ADR は設計判断の背景、比較、決定、利点と制約を残す。「採用」は方針の決定であり、実装済みを意味しない。
- 文書の要件は実装時の契約とする。未決定事項を実装前に対応 Issue で具体化し、検証結果は Issue・PR に記録する。

## 変更前に読む文書

| 文書 | 対象となる変更 | 状態 |
| --- | --- | --- |
| [データ・調査仕様](specs/data-and-research.md) | 保存、収集、分析、レポート、ジョブと実行上限 | 基本要件を定義した。永続保存方式・分析の上限などは未決定、収集・JSON 出力は導入した |
| [公開デモ仕様](specs/public-demo.md) | 公開 UI、認証、利用期限、実行制限、README | MCP は認証なしの方針。バックエンド分析の利用コードなどは未決定、デモは未配置 |
| [最小 HTTP サーバー仕様](specs/http-server.md) | healthz、PORT、開発用の公開ポート選択、サーバーの期限・終了処理 | Issue #11 で実装した。ローカル検証結果は作業記録へ記載する |
| [Steam レビューの1ページ取得](specs/steam-review-page.md) | Steam HTTP adapter、取得条件、応答と通信の上限 | Issue #17 で実装した。収集操作の契約は別仕様、保存は未導入、収集操作を MCP へ接続した |
| [Steam レビューの上限付き収集](specs/collect-reviews.md) | 複数ページ収集、条件・重複・上限・途中失敗 | Issue #21 で実装した。MCP 接続を導入した。永続保存・再開・Web 接続は未導入 |
| [収集 CLI と保存先](specs/review-cli.md) | CLI の入力・終了・JSON・保存先の契約 | Issue #23 でローカル CLI と JSON 保存実装を導入した。永続保存は未導入 |
| [レビュー収集 MCP](specs/review-mcp.md) | ローカルの MCP 接続、入力・応答・期限・部分失敗 | Issue #25 でローカル MCP を導入した。SDK と ChatGPT の確認を区別して記録する |
| [レビュー分析仕様](specs/review-analysis.md) | クライアント AI 向けガイドの提供・内容・出力と保証範囲 | Issue #35 で版付きの案内を導入した。分析生成・意味検証はバックエンドに導入しない |
| [分析の利用手順](review-analysis.md) | 依頼テンプレート・深掘り・結果の確認 | [架空データの例](examples/review-analysis.md) を併記した。実 AI の動作検証とは区別する |
| [MCP 利用制限](specs/mcp-usage-limits.md) | MCP の開始回数・固定窓・同時実行数・設定・拒否 | Issue #27 で1プロセスの制限を導入した。共有予算・認証は未導入。公開配置の基本通信は確認済み |
| [認証なしの公開 MCP](specs/mcp-public-access.md) | 公開 URL、許可 Host、匿名公開 | Issue #29 で URL と Host 保護を導入した。ユーザー配置の基本 HTTP 通信は成功、実 ChatGPT 接続は未確認 |
| [Render 配置基盤](specs/render-deployment.md) | イメージ公開・digest・Render 設定・公開接続検査 | Issue #31 で配置準備を導入した。ユーザー配置の基本通信は成功、digest 経路の実配置は未実施 |
| [ユースケース](use-cases.md) | 操作全体の一覧・責務と adapter・入口の境界 | 収集と収集・保存操作を導入した。CLI と MCP を接続した。Web は未接続 |
| [ADR-0001：Go・Docker・Render による実行・配置](adr/0001-runtime-and-deployment.md) | 実行環境、開発・検証用 Docker、本番用 CI、Render 配置 | 採用。Compose・本番用イメージと検証 CI、配置経路の準備を追加した。digest 配置は未確認 |
| [開発手順と規約検査](development.md) | 規約バリデーター、Git フック、Issue・PR の作成手順とテンプレート、CI | ローカル検査と PR・Issue イベントの規約 CI を確認した |
| [Render 配置手順](deployment.md) | GHCR・生成設定・公開接続・無料枠と資格情報 | 手順を導入した。registry 送信と digest 配置は未実行 |
| [Release 手順](releases.md) | Draft の作成・公開、本文・タグ・対象・CI の事前検査 | Issue #5 で CLI と通常テストを追加した。実 CI・公開結果は Issue に記録する |
| [移植ガイド](reuse.md) | 他のプロジェクトへの規約・検査・フック・CI の導入 | v0.0.1 の移植対象と調整箇所を記録した |
| [ADR-0002：規約検査の実行環境](adr/0002-conventions-tooling.md) | Python、検査ツール、フック・CI の責務 | 採用。保護設定は現時点の契約では利用できず未設定 |
| [ADR-0003：Go の整形・リンター](adr/0003-go-format-and-lint.md) | Go の整形、リンター、検証用 Docker、フック・CI | 採用。固定版の解析と index の整形検査を導入した。実 CI の結果は Issue・PR に記録する |
| [ADR-0004：Steam の公開レビュー API](adr/0004-steam-review-api.md) | Steam API の選択、引数形式、取得 adapter の上限と再試行の責務 | 採用。実装とローカル検証は Issue #17 の作業記録へ記載する |
| [ADR-0005：上限付き収集と部分結果](adr/0005-bounded-review-collection.md) | 収集操作の依存・上限・途中失敗・重複の判断 | 採用。配置を ADR-0013 で部分置換した。実装と検証は Issue #21・#56 に記録する |
| [ADR-0006：収集結果の保存先](adr/0006-collection-output-boundary.md) | 保存先の注入、JSON の境界、部分結果の保存期限・失敗 | 採用。配置を ADR-0013 で部分置換した。CLI と JSON adapter を導入し、DB は未導入 |
| [ADR-0007：ローカルのレビュー収集 MCP](adr/0007-review-mcp.md) | SDK、プロトコル、短時間の取得と分析の担当 | 採用。実装・検証の状態は Issue #25 の作業記録へ記載する |
| [ADR-0008：MCP 利用制限](adr/0008-mcp-usage-limits.md) | 制限の状態所有・受付・返却・固定窓と保証範囲 | 採用。1プロセスで管理する。再起動・複数台の予算保証は持たない |
| [ADR-0009：認証なしの公開 MCP](adr/0009-public-mcp-access.md) | 公開設定と Host 保護、OAuth を導入しない理由 | 採用。配置と実接続は未確認 |
| [ADR-0010：Render 配置基盤](adr/0010-render-deployment.md) | 手動の GHCR 公開・Free・資格情報と配置準備の範囲 | 採用。実 CI・registry・配置は未実施 |
| [ADR-0011：クライアント向け分析ガイド](adr/0011-client-review-analysis-guide.md) | MCP 結果へのガイド添付と分析の担当・制約 | 採用。版付きの本文を埋め込み、クライアントの最終回答は強制しない |
| [ADR-0012：Steam の名前付き数値型](adr/0012-steam-domain-types.md) | Steam の ID・時刻・API 列挙値、JSON 境界との変換 | 採用。実装・検証は Issue #43 と作業記録に記載する |
| [ADR-0013：各領域内のレイヤー分類](adr/0013-domain-folder-layers.md) | 既存領域を維持した domain・application・adapter の配置・依存 | 採用。各領域内の分類と依存検査を導入した。[検証記録](worklogs/56-domain-folder-layers.md) に実施結果を残す |
| [ADR-0014：Steam の収集状態](adr/0014-steam-collection-state.md) | 収集状態・重複・カーソル・終了と取得手順の境界 | 採用。収集状態と取得手順を分離した。[検証記録](worklogs/58-steam-collection-state.md) に実施結果を残す |
| [ADR-0015：収集条件の生成と利用](adr/0015-collection-options-boundary.md) | 未検証の設定・有効な条件・ゼロ値・結果のコピーの境界 | 採用。収集条件の生成と利用を分離した。[検証記録](worklogs/60-collection-options-boundary.md) に実施結果を残す |
| [ADR-0016：MCP の応答完了と同時枠](adr/0016-mcp-response-slot-lifetime.md) | 結果処理の同期実行と同時枠の保持・返却、レイヤー間の連携 | 採用。応答変換完了まで枠を保持する構成に修復した。[検証記録](worklogs/62-mcp-response-slot-lifetime.md) に実施結果を残す |

変更が複数の範囲にまたがる場合は、対応する文書をすべて読む。参照した節と ADR、要件に対応する検証を Issue・PR に記録する。ADR を追加・置換するときは、この一覧と適用範囲を更新する。

## 現在の導入状況

Issue #11 で Go モジュール、最小 HTTP サーバー、Dockerfile と Compose を追加した。DB・バックエンドの分析・公開デモは未導入とする。ローカル検証と新 CI の実行状況は [Issue #11](worklogs/11-http-server.md) に記録する。Issue #3 で Python の規約ツール・Git フック・テンプレート・CI の設定を追加した。PR #4 と Issue #5 の規約 CI を GitHub 上で確認した。Issue #5 で Release CLI と移植ガイドを追加した。保護設定は未設定とし、検査の成功だけでマージの機械的な制限や AI の遵守を保証しない。対象 SHA と結果は [Issue #3](worklogs/3-conventions.md) と [Issue #5](worklogs/5-release.md) の作業記録に残す。

Issue #17 で [Steam レビューの1ページ取得 adapter](specs/steam-review-page.md)、Issue #21 で [上限付き収集操作](specs/collect-reviews.md) を追加した。Web の入口への接続と永続保存は未導入とし、[1ページ取得の記録](worklogs/17-steam-review-page.md) と [収集の記録](worklogs/21-collect-reviews.md) にローカル検証を残す。

Issue #23 で [収集 CLI と保存先の境界](specs/review-cli.md) を追加した。JSON 変換と writer への出力を分離し、取得と CLI の入力処理は保存先の共通契約を使う。検証は [作業記録](worklogs/23-review-collection-cli.md) に残す。

Issue #25 で [レビュー収集 MCP](specs/review-mcp.md) を導入した。分析は接続元の ChatGPT が担当する。ローカルの SDK・本番イメージ検証と ChatGPT アプリの手動確認の状態は [作業記録](worklogs/25-review-mcp.md) に分けて残す。

Issue #27 で [MCP 利用制限](specs/mcp-usage-limits.md) を追加した。登録済み MCP ツールの実呼び出し・取得・分析を確認した。検証と公開前の制約は [作業記録](worklogs/27-mcp-usage-limits.md) に残す。

Issue #29 で [認証なしの公開 MCP](specs/mcp-public-access.md) を追加した。公開 URL と許可 Host・Origin の検査を導入し、OAuth は採用しない。[作業記録](worklogs/29-public-mcp-access.md) にローカル検証と公開配置・実接続の未実施を記録する。

Issue #31 で配置準備を追加した。ユーザー配置の公開 URL で基本通信を確認し、実 registry 送信・digest 配置・実 ChatGPT 接続は未確認とする。[作業記録](worklogs/31-render-deployment.md) に区別して記録する。

Issue #33 で評価種別の指定を追加した。Go・CLI・MCP で共通の条件を使い、全評価の省略互換性を維持する。[作業記録](worklogs/33-review-type.md) にローカル検証と未配置の範囲を記録する。

Issue #35 でレビュー収集結果へ版付きの分析ガイドを添付した。[利用手順](review-analysis.md) と架空レビューの説明例を追加し、バックエンドの分析生成・意味検証とは区別する。[作業記録](worklogs/35-review-analysis-guide.md) に確認した契約と未確認の範囲を記録する。

Issue #40 で MCP の失敗・結果変換を収集の実行手順から分離した。公開契約を維持し、[作業記録](worklogs/40-mcp-result-conversion.md) に検証結果と未実施のコード CI・配置を記録する。

導入が進んだら、対応 PR でこの状態を更新する。`docs/use-cases.md` は操作の追加・変更時に更新する。自動生成の仕様目次は別 Issue で形式と検査を決定してから導入する。
