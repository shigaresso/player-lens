# Issue #17：Steam レビューの1ページ取得

## 対象と参照

[Issue #17](https://github.com/shigaresso/player-lens/issues/17) を origin/main `8e6ceb5` から `codex/17-steam-review-page` で進めた。文書一覧、データ・調査仕様のレビューと収集条件・ジョブと上限、開発手順、採用済み ADR-0001・0002・0003 を参照した。契約は [1ページ取得仕様](../specs/steam-review-page.md) と [ADR-0004](../adr/0004-steam-review-api.md) に記録した。

## Red・Green

関連テストは `docker compose run --rm --no-deps -T verify go test -count=1 ./internal/steam` で実行した。初回のみ `--build` を付けた。

- 正常・空ページの取得条件、文字列 ID、本文と時刻、特殊文字のカーソルは、非テスト Go ファイルがないコンパイル失敗で Red、Client と型・取得処理の導入後に Green となった。
- 不正入力と接続設定の拒否は、`ErrInvalidArgument` がないコンパイル失敗で Red、入力検査とエラー定義の導入後に Green となった。
- HTTP 429・500・302・204、非再試行・非リダイレクト、渡された HTTP クライアントの保持は、`HTTPStatusError` がないコンパイル失敗で Red、型とリダイレクト拒否の導入後に Green となった。
- JSON・wrapper・ID・件数・カーソルの不正応答は `ErrInvalidResponse` 未定義で Red、検査導入後に Green となった。JSON の構文エラーも `errors.As` で取得できた。
- 2 MiB超過の応答が成功する Red を確認し、読込を上限+1バイトまでに制限して超過を拒否し Green となった。2 MiBちょうどの成功も確認した。
- 期限なし・長い期限の通信で最大10秒の期限がない Red を確認し、派生 context の期限と終了時の cancel 導入後に Green となった。呼び出し側の短い期限・キャンセル・期限切れは既存の伝播を維持した。
- 読取失敗・HTTP 失敗に Close 失敗が重なると元の原因を失う Red を確認し、原因を結合して Green となった。成功・各失敗の Body を一度閉じ、HTTP 失敗で本文を読まず、サイズ超過でも読込を打ち切った。
- 入力の最小・最大、公式接続先とキーを送らない条件、Transport の失敗は実装済みで Green となり、回帰テストだけのコミットにした。人工的な Red は作らなかった。

## 実 API の読み取り

2026-10-08 に AppID 440、英語、Recent、1件を匿名で1回読み取り、HTTP 200、`response` wrapper、文字列 ID と整数時刻を確認した。本文をログへ出力せず、外部データをリポジトリへ保存しなかった。実接続は手動の確認で、通常テスト・CI の対象にはしていない。

## 最終検証

`bash scripts/check-app.sh` が成功した。整形、go vet、通常テスト、golangci-lint、`go test -tags=integration -race -count=1 ./...` と本番用イメージの実検査を実行した。リンターでサイズ超過エラー文字列の ST1005 を検出し、小文字の先頭へ修正後に0件となった。結合タグの実行には通常の steam テストも含まれる。

本番イメージ `sha256:74fc2ec8fcaeee2f910e45035718303e606aa16a6b3ab924a45781c29665e3d5` で PORT 8080・10000 の HTTP 応答、SIGTERM・SIGINT の終了、不正 PORT の非0終了を確認した。今回はサーバーの入口へ adapter を接続していないため、サーバーのバイナリは変更前と同じとなった。

`docker compose config --quiet` が成功した。`docker run --rm --mount type=volume,source=player-lens_go-build,target=/go-cache player-lens-verify go test -count=1 ./internal/steam` でホストのソース共有なしでもテストが成功し、Docker の COPY 対象に internal が含まれることを確認した。Compose Watch は構成を検査し、継続監視の実操作は今回行っていない。

コミットの必須 pre-commit で共通144テスト、Lefthook・actionlint・空白・シェル構文と Go の index 整形を確認し、commit-msg で日本語の本文付きメッセージを検査した。成功済みの検査を手動で重ねず、必須フックは回避しなかった。

AI によるセルフレビューで入出力、固定条件、期限・読込上限、HTTP クライアントの保持、ゼロ値、原因の保持と資源の解放、Docker の許可対象と秘密情報・生成物が差分へ入っていないことを確認した。Render 配置、保存、複数ページの収集、MCP、coverage・分岐・mutation の目標達成、人によるレビューは未実施とする。push・PR 作成・Issue クローズは行わず、このブランチの実装に対する GitHub CI の結果も未確認とする。
