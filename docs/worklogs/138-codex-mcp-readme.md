# Issue #138：Codex の MCP 設定と分析依頼例

## 対象と変更

- Issue：https://github.com/shigaresso/player-lens/issues/138
- 開始時の origin/main：34de6e3d6dc0cca205d3972c084701b29cd6769b。
- ブランチ：codex/138-codex-mcp-readme。未コミットの変更がないことを確認し、fetch 後の origin/main から作成した。
- README に、ローカルの起動・実ポートでの登録と、Render の公開 URL の登録を分けて記載した。設定画面と CLI の両方を案内し、接続名を player_lens_local と player_lens_render に分けた。
- 設定後の再起動・ツール確認、接続先ごとの少量取得と分析の依頼例、停止・ポート変更・接続失敗・利用制限時の確認を追加した。
- 詳細な分析手順から README の設定へリンクし、公開デモ仕様の README 要件と整合させた。
- 追加の確認で、ポートの表示先を起動コマンドを実行したターミナルと明記した。scripts/dev.py の出力に合わせて「開発サーバー」の表示例を載せ、表示中の8081から登録先の /tools URL を作る例を補足した。開発環境の説明も同じ表示へ揃えた。

## 参照と判断

- docs/README.md「変更前に読む文書」と docs/specs/public-demo.md「README の要件」「期待結果と検証への対応」を参照した。
- docs/specs/review-mcp.md「接続と入力」「結果・失敗・資源」と ADR-0007「決定」により、/tools・collect_reviews・入力上限と、収集は Go・分析はクライアントという既存の境界を維持した。
- docs/specs/review-analysis.md「分析として案内する内容」「保証する範囲」、ADR-0011「決定」、docs/review-analysis.md「依頼の例」と、埋め込んだ analysis_guide.md を参照し、根拠 ID・意見と解釈の区別・偏り・部分失敗・外部指示を扱う例にした。
- docs/specs/mcp-public-access.md「HTTP の受付」、ADR-0009「決定」、docs/mcp-debug.md「起動と登録」「少量の手動確認」、docs/deployment.md「公開後の確認」を参照した。匿名公開の設定と利用枠は変更しなかった。
- docs/development.md「Issue・PR の作成手順」「各フックの責務」と ADR-0002「決定」に従い、Issue は事前検査付き CLI で作成した。
- 2026-11-01 JST に [OpenAI 公式 MCP 設定手順](https://learn.chatgpt.com/docs/extend/mcp) と [CLI リファレンス](https://learn.chatgpt.com/docs/cli/reference) の本文を確認した。設定画面での Streamable HTTP 登録・保存後の再起動と、CLI の add --url・list に照合した。

登録内容の確認、通信診断、Codex による実際のツール呼び出しと分析を区別した。既存の実接続記録は保持し、今回の文書確認を新しい取得・分析の成功として記載しなかった。

## 検証

| 対象 | 実行・確認 | 結果 |
| --- | --- | --- |
| CLI の構文 | codex mcp add --help、codex mcp list --help | NAME・--url と登録一覧の構文を確認した。設定を変更するコマンドは実行しなかった |
| 依頼例 | protocol/schema.go・input.go、Steam の条件と既存仕様への照合 | 両例とも app_id=440、language=all、review_type=all、page_size=3、max_reviews=3、max_pages=1 が既存契約内であることを確認した |
| 文書の参照 | Python 標準ライブラリによる変更文書3ファイルの相対リンク・見出し照合 | 38件すべてのファイル・見出しを確認した。変更前の README 見出しを指すリンクも検索し、該当なしだった |
| 記載内容 | 登録コマンド2件・依頼例2件の照合と差分のセルフレビュー | 接続名・URL・分析ガイド・根拠 ID・取得結果・追加取得なしの対応を確認した。端末ユーザー名を含むホームパスと実レビュー・認証情報を記載しなかった |
| 共通検査 | bash scripts/check.sh | Python 184件、差分の空白、Lefthook 設定、actionlint、シェル構文が成功した |

文書だけの変更として、新しいテストや形式のための Red は追加しなかった。Go の実装・設定・実行環境は変更していないため、check-app.sh とフック実装自体の verify_hooks.py は再実行しなかった。coverage 閾値・分岐・mutation は未導入であり、実行していない。コミット時には必須の pre-commit・commit-msg を実行し、その結果は Issue に記録する。

## 未実施と範囲

Codex の設定変更・再起動・画面操作、実 Steam の取得・AI 分析、Render の配置や実行バージョン照合は今回行っていない。手順の照合と実クライアントでの確認を区別する。アプリケーションの動作、認証、実行制限、保存先、生成データは変更せず、新しい I/O や依存を追加しなかった。push・PR 作成・Issue のクローズをせず、ローカルコミットで止める。
