# Issue #33：評価種別を指定するレビュー収集

[Issue #33](https://github.com/shigaresso/player-lens/issues/33) に対応し、最新 origin/main `b4957cde3b575b61299a586065db7bdb2966fb48` から `codex/33-review-type` を作成した。upstream は未設定。既存の未追跡 `.DS_Store` は保持してコミット対象から除いた。push・PR 作成・Issue クローズは行っていない。

## 参照した仕様と判断

文書一覧、[データ・調査仕様](../specs/data-and-research.md) の「レビューと収集条件」「分析とレポートの根拠」、[1ページ取得](../specs/steam-review-page.md) の「API と入力」、[収集](../specs/collect-reviews.md) の入力・走査・部分失敗、[CLI](../specs/review-cli.md) の入力・JSON、[MCP](../specs/review-mcp.md) の入力・結果、[利用制限](../specs/mcp-usage-limits.md) の受付順序、公開デモ仕様・開発手順と ADR-0004〜0008 を参照した。

[ADR-0004](../adr/0004-steam-review-api.md) に評価種別の採用理由を追記した。取得後の間引きを避け、Steam の review_type に条件を渡す。公開する文字列と API の数値を分離し、PageOptions の正規化を CollectionOptions・CLI・MCP で再利用する。空は全評価を表す all に正規化し、型だけで有効性を保証するとは扱わない。

公式仕様は2026-10-23に [GetAppReviews](https://partner.steamgames.com/doc/webapi/IUserReviewsService#GetAppReviews) と [評価種別](https://partner.steamgames.com/doc/webapi/IUserReviewsService#EUserReviewsReviewType) を確認した。All=0、Positive=1、Negative=2に対応した。既存の Recent・購入種別・off-topic 条件、件数・時間・回数・同時数と再試行なしは維持した。

JSON の options.review_type は schema_version=1 への追加項目とした。旧 JSON の欠落は従来の全評価を意味し、既存項目の意味を変えない。永続的な根拠保存・本文版・バックエンド AI を導入したとは扱わない。

## Red・Green とコミット

| コミット | 契約と実際の Red | Green の確認 |
| --- | --- | --- |
| `e7a0660` | ReviewType と PageOptions / CollectionOptions のフィールドが未定義でコンパイル失敗 | 共通正規化の有効3種・空・不正値・入力非変更・再適用を確認し、通常テスト全体が成功した |
| `0664f95` | positive / negative でも API の review_type が0だった | 数値1 / 2の指定、省略・全評価の互換性、返却する推薦フラグと条件を確認し、steam のテストが成功した |
| `d6c3dbd` | 第1サイクルはページ要求の評価種別が空で失敗。第2サイクルは JSON の options.review_type が欠落して失敗 | 各ページへの条件伝播・部分失敗を Green にしてから JSON のサイクルへ進み、空・正常・部分失敗を確認した。関連する steam / jsonoutput / reviewmcp のテストが成功した |
| `7e8ada1` | --review-type が未知の flag として拒否され、help にもなかった | 指定・空・省略・不正値、取得と保存への接続、integration タグを含む CLI 実プロセスの終了コードを確認した |
| `609779c` | review_type 入力が未定義項目として拒否され、入力 schema にもなかった | parseInput と schema の選択肢・既定値・任意項目の契約、null・型違い・未知値の拒否を確認した |
| `eb0206c` | 既存実装で Green のため、意図的な失敗は作らなかった | 実 SDK 4版・不正入力の非消費・利用上限の拒否・部分失敗の条件保持、HTTP の通信前拒否を race 付きで確認し、テストだけをコミットした |

テスト追加後に意図した失敗を確認してからプロダクションコードを変更した。正規化結果に all が明示されるため、既存テストの既定条件の期待結果も更新した。Red のまま共有するコミットは作らなかった。

## 全体検証

- `bash scripts/check-app.sh` が成功した。gofmt、vet、通常 Go テスト、golangci-lint（0 issues）、Git 情報取得失敗・Git 不在・checkout 不在の3条件、integration タグを含むテストと race detector、本番イメージの HTTP/MCP・設定・終了を確認した。integration コマンドは通常テストも含む。
- 本番検証の image ID は `sha256:92df7904a58b3f874d223a3dca037553742b2d9149290f045fd1f252bfbde3b0`。ホスト側 Go やソース共有に依存せず、同じイメージで異なる PORT・SIGTERM/SIGINT・匿名公開と Host/Origin 保護・不正設定を確認した。registry 公開・Render 配置の成功を意味しない。
- 各コードコミットの必須 pre-commit・commit-msg が成功した。Python は174テスト、actionlint と差分の空白検査が成功した。新しい Python ツール・依存は追加していない。
- `python3 scripts/verify_hooks.py` の8テストが成功した。隔離環境のフック検証と、本物の Docker によるアプリケーション検証を区別した。実 push は行っていない。今回フックのコードは変更していない。
- Go・MCP の関連テストと通常の全体検証は実 Steam・LLM を呼ばず、HTTP は httptest / Transport、MCP は実 SDK と差し替えた取得依存で確認した。
- 実装・差分を AI がセルフレビューした。人によるレビューは未実施。coverage 閾値・分岐網羅・mutation は未導入で、Stryker は実行していない。

## 実 Steam の少量確認

2026-10-23、変更したローカル CLI から次のコマンドを1回だけ実行した。

```sh
docker compose run --rm --no-deps -T collect --app-id 4867330 --language all --review-type negative --page-size 1 --max-reviews 1 --max-pages 1
```

終了コード0、App ID 4867330、options.review_type=negative、取得1件・成功ページ1、stop_reason=review_limit、レビュー ID `237092522`、recommended=false を確認した。API の1リクエストだけで、追加収集・自動再試行・全文の検証記録への転載は行わなかった。応答は Steam のキャッシュの影響を受け得るため、常に最新という保証は付けない。

これは CLI → Steam の手動確認であり、Render や ChatGPT の登録ツール経由で新機能を確認したものではない。MCP の新しい入力はローカルの実 SDK 4版で確認した。Render の構成・配置経路とアプリ設定は変更していない。

## 残る確認と対象外

このブランチは未 push のため、対象 SHA の GitHub CI・Render 配置・公開 MCP の評価種別指定・ChatGPT アプリ経由は未確認である。既存の公開サーバーの動作をローカル変更の確認へ流用しなかった。利用例は [MCP デバッグ](../mcp-debug.md) に記載した。

バックエンド AI・LLM API キー・分析用 Web 画面・DB・レポート生成・別の収集条件は追加していない。保存先の注入と CLI / MCP の共通操作の境界を維持した。
