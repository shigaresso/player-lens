# Steam ページ取得に応答判断と Body の所有をまとめた

[Issue #129](https://github.com/shigaresso/player-lens/issues/129) に対応した。PR #128 統合済みの origin/main（263cb54）から codex/129-fetch-response-ownership を作成した。

FetchReviewsPage に HTTP ステータスの判断と Body の Close を移した。Do の通信エラー確認、Close の予約、HTTP 200 の確認、本文読み取りの呼び出しが順に読めるようにした。取得全体の成功条件と資源管理を同じ操作に残すための責務移動である。

readReviewPageResponse とそのファイルを readReviewPageBody・read_review_page_body.go へ置き換えた。io.Reader・正の App ID・正規化済み条件を受け取り、上限付き読み取りと decodeReviewPage の呼び出しだけを担当する。HTTP Response・ステータス・Close・context の所有を渡さず、旧名のラッパーは残していない。

FetchReviewsPage の戻り値に page・err の名前を付け、Close 失敗時にはページをゼロ値にして元の原因と併せて返す処理をそのまま移した。defer cancel より後に Body の defer を登録することで、Close の後に取得用 context を解放する順序を維持した。名前付き err を使用するため、条件の正規化は宣言から代入へ変更した。

## 検証

動作を変えない責務移動として、既存テストを変更せず使用した。形式的な Red や重複テストは追加していない。変更前に固定版 Compose verify で go test -count=1 ./internal/steam/adapter が成功した。

変更後の共通検証の初回は、移動した複数行の defer の前に必要な空行がなく、wsl_v5 の after-block が失敗した。空行だけを修正し、検査の除外や設定変更は行わず再実行した。

bash scripts/check-app.sh は終了コード0で成功した。整形・go vet ./...・go test -count=1 ./...、通常と integration タグの lint（0 issues）、Git 非依存3条件、go test -tags=integration -race -count=1 ./... を確認した。最後のコマンドは通常・結合の両方のテストを含む。

既存の BodyLifecycle の10ケースで、成功・HTTP 不正・読み取り・サイズ・JSON の失敗、Close とそれぞれの複合失敗、原因の保持・ページのゼロ値を確認した。Body を1回閉じ、HTTP 不正時は本文を読まず、Close 後に取得用 context を解放する契約を維持した。BodyPanic の既存テストで同じ panic の伝播・Close の1回実行・context の解放順序を確認し、他の HTTP・サイズ境界・入力・Context・ページ変換のテストも成功した。

本番用 image ID sha256:3b24eaa5b4678b43d1f0e318f8fa42f7e86d4d4805e1494a5b73f3e52610c274 を一度ビルドし、同じイメージで HTTP/MCP、ローカル／公開の Host・Origin、PORT、SIGTERM／SIGINT、不正な公開 URL・PORT・利用制限設定の非0終了を確認した。ソース共有なし・非 root の既存検証経路を使用した。

ステータス判断と Close のブロックは旧実装と同一で、本文処理は io.Reader の受け口以外を維持したことを照合した。既存の全テストファイルは変更していない。git diff --check、旧名の参照漏れ、文書リンク、端末固有情報の非混入、差分のセルフレビューを確認した。必須コミットフックの結果と確定コミット・範囲の規約検査は Issue に記録する。

## 参照と確認範囲

文書一覧、1ページ取得仕様の入力・リクエスト構築・結果と失敗・応答解釈・検証、ADR-0004 の Body の所有、ADR-0012 の型と変換の境界、操作一覧、データ・調査仕様の収集と導入状況、開発手順と既存の検証経路を参照した。実装に先立ち仕様と ADR-0004 の Issue #66 の配置判断を、ユーザーが合意した新しい所有方針に整合させた。過去の作業記録は保持した。

GitHub 認証・Docker・Git の書き込みは許可された権限で実行し、ログはリポジトリ外の一時ディレクトリへ置いた。新しい保存先・実データ・依存は追加していない。

push・PR 作成・Issue のクローズ、GitHub CI・実 Steam／LLM・Render の検証、人によるレビューは未実施。coverage 閾値・分岐網羅・mutation（Stryker を含む）は現行では未導入であり、今回の確認結果に含めない。
