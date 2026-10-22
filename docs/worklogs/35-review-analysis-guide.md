# Issue #35：クライアント向けのレビュー分析ガイド

## 背景・目的

[Issue #35](https://github.com/shigaresso/player-lens/issues/35) に対応し、最新の origin/main `ba46d6e56747b9ab6682eaf86afd5f2f0a8352cf` から `codex/35-review-analysis-guide` を作成した。レビューの良い点・悪い点の列挙から、何が、どの条件で、なぜ評価されるかを読み、人間が設計の選択肢を検討できるようにした。

最初のローカル実装完了時点では未 push・upstream 未設定とし、PR 作成・Issue のクローズを行わず停止した。その後に実施した登録済みクライアントの確認を末尾に追記した。

## 変更内容

- MCP の共通結果へ `analysis_guide.version="1"` と `instructions` を追加した。単一の Markdown をバイナリへ埋め込み、正常・空・不正入力・部分失敗・利用制限・応答量超過で同じガイドを返した。ツール説明と出力 schema も対応させた。
- 本文からの分類・意見の抽出と統合・条件による賛否・レビュー ID・意見と解釈の区別・作品間の比較・選択肢と影響を案内した。分析はクライアント AI、採否は人間に残した。
- ガイドも二重化した JSON の128 KiB制限へ含め、本番イメージ検査で版・本文・構造化結果とテキストの一致を確認した。Collection の schema_version=1、CLI、利用制限、再試行なしの責務を維持した。
- [利用手順](../review-analysis.md)、コピー用の依頼と深掘り例、[架空レビューの分析例](../examples/review-analysis.md)、仕様と ADR、文書一覧と AGENTS の参照を追加した。
- `.DS_Store` を Git と Docker の対象から配下も含めて除外した。既存のファイルは削除していない。

### Red・Green とコミット

| コミット | 確認した契約・失敗 | Green と対応 |
| --- | --- | --- |
| `7c700e5` | `.DS_Store` が未追跡のまま表示され、除外設定がなかった。文書・設定変更なので形式的な Red は作らなかった | git check-ignore がルート・既存配下・さらに深いパスを除外した。git ls-files の対象は0件だった |
| `c119c5f` | 応答付与の10状況が「版1の分析ガイドがない」で Red。実装後に通常テストで Green。次に実 SDK の対応4版が「分析ガイドの存在を説明していない」で Red | schema と説明を追加した後、通常テストと integration タグ付き MCP パッケージが Green。すべての結果を読む共通ヘルパーでガイドと JSON テキストの一致を確認した |
| `e663301` | 本番イメージ検査がガイド欠落・版違い・本文違い・型違い・JSON 不一致を見逃し、5状況で Red | 検査追加後に Python の関連6テストが Green。容量63000文字の境界は既存の計測で最初から Green だったため、失敗を作らず回帰テストにした |
| `d1eb21e` | 開発イメージの実読み取りで cmd・internal 配下の .DS_Store が残る失敗を確認した | `.dockerignore` に再帰パターンを追加した。再ビルド後のソース共有なしのイメージで find の結果が空となり、除外を確認した |

本番イメージ検査のテストは、最初の実行でテスト用 HTTP 応答の getheader が不足していた。ダブルを修正し、ガイドを見逃すという意図した Red を再確認してから検査の実装へ進んだ。Red のままのコミットは作っていない。文言の存在検査だけで AI の意味解釈を保証したとは扱わない。

## 検証

- `bash scripts/check-app.sh` が成功した。Docker 上の gofmt、vet、通常 Go テスト、golangci-lint（0 issues）、Git 取得失敗・Git 不在・checkout 不在、integration タグを含むテストと race detector、本番イメージの HTTP/MCP・設定・正常終了を確認した。integration の実行には通常テストも含む。
- この全体検証の本番 image ID は `sha256:182608bf067c1c214c1e32323a29bfbf9f34498eec1eb0e7779f0eda3ad7f2f0`。同じイメージをソース共有なし・非 root・読み取り専用で起動し、埋め込んだガイド、異なる PORT、SIGTERM/SIGINT、匿名公開と Host/Origin 保護、不正設定の拒否を確認した。
- 再帰的な Docker 除外の修正後も production を再ビルドし、上記と同じ image ID であることを確認した。コード・本番成果物が同じため、成功済みの動作検証は重ねなかった。
- 各コミットの必須 pre-commit・commit-msg が成功した。共通検査は Python 177テスト、Lefthook 設定、actionlint、シェル構文、差分の空白を確認した。Go の変更は index の整形検査も通した。`bash scripts/check.sh` は必須フックから実行した結果を用い、同じ手動実行を重ねなかった。
- `git check-ignore` と `git ls-files` で `.DS_Store` の除外・未追跡を確認した。開発イメージの再ビルドと `docker run --rm --read-only --entrypoint find player-lens-verify /src -name .DS_Store -print` で配下にも含まれないことを確認した。
- 通常テスト・SDK・イメージ検証は実 Steam・実 LLM を呼ばなかった。今回、追加の実レビュー取得は行っていない。前の会話での実分析は新ガイドの実クライアント検証へ流用していない。
- AI が差分・担当の境界・ガイドと例の説明をセルフレビューした。人によるレビューは未実施。フック・CI 自体は変更しておらず、隔離した verify_hooks は今回再実行していない。実 pre-push は push しないため未実行とした。

### 残る確認と制約

最初のローカル実装完了時点では、このブランチの GitHub CI・Render 配置・登録済みクライアントからの新ガイド受信と最終分析の確認は未実施だった。後述の手動確認ではローカルの登録済みクライアント経由を確認した。実 SDK による応答確認と、外部クライアントの AI の遵守・分析品質を区別する。厳密に追加フィールドを拒否する独自クライアントは更新が必要で、ガイドの分だけ本文へ使える応答量は減る。

coverage 閾値・分岐網羅・mutation は未導入で、Stryker は実行していない。バックエンド LLM・DB・Web UI・分析結果の意味検証・ランキング自動取得は追加していない。

## 参照仕様・ADR

- [文書一覧](../README.md)、[データ・調査仕様](../specs/data-and-research.md) の「分析とレポートの根拠」「ジョブ・上限・部分失敗」。
- [MCP 仕様](../specs/review-mcp.md) の「結果・失敗・資源」「デバッグと検証」、[公開デモ仕様](../specs/public-demo.md) の「README の要件」。
- [ユースケース](../use-cases.md) の「パーツとの境界」、[開発手順](../development.md) の「Go・Compose・本番用イメージの検査」「本文検査の契約」「Issue・PR の作成手順」。
- [ADR-0001](../adr/0001-runtime-and-deployment.md) の「決定」、[ADR-0002](../adr/0002-conventions-tooling.md) の検査の責務、[ADR-0007](../adr/0007-review-mcp.md) の「決定」「利点と制約」。
- 新規の [レビュー分析仕様](../specs/review-analysis.md) と [ADR-0011](../adr/0011-client-review-analysis-guide.md)。
- [MCP Tools 公式仕様](https://modelcontextprotocol.io/specification/2025-11-25/server/tools) の Structured Content と Output Schema を2026-10-23に参照し、サーバー結果の形式と AI の最終回答を区別した。

## 関連 Issue

Closes #35

Issue #33 は PR #34 で main へ統合済み。Issue のクローズ操作はしていない。

## 登録済み MCP からの手動確認

2026-10-23、ユーザーの依頼により、コミット `0d284aa` のソースでローカルの app コンテナを再ビルド・起動した。接続先は登録済みの `http://127.0.0.1:8080/tools` とし、Render や独自 HTTP クライアントの代用は使わなかった。

登録済み `collect_reviews` へ App ID 4950470（Tidy Up Together）、language=all、review_type=positive / negative、page_size=10、max_reviews=10、max_pages=1 を指定した。Steam への取得は直列の計2回で、おすすめ10件・非おすすめ10件、両方とも1ページ・stop_reason=review_limit、error_code は空・omitted_reviews=0だった。両応答の analysis_guide.version=1 と本文の受信を確認した。

受け取ったガイドに沿い、物量・自由配置の情報不足・成長の効果・長時間の音楽などを本文から分類し、根拠 ID と条件を対応させて設計の選択肢・得失を整理した。例えば、物量への好評は 237833223・237908479、配置のやり直しへの不満は 238027661・237528769・237504315、成長と作業量の関係は 237908479・237866742・238299359 を根拠にした。全文レビューは本書へ転載していない。

これは当該クライアントと今回の入力における手動確認であり、他の AI がガイドを守ることや、分類・解釈の意味が常に正しいことは保証しない。抽出した両評価の比率を全体の好評率には使っていない。通常テスト・CI の実 Steam 接続は追加せず、公開 Render での新ガイドの確認は未実施のままとした。
