# Issue #23：収集 CLI と保存先の境界

## 範囲と参照

[Issue #23](https://github.com/shigaresso/player-lens/issues/23) を最新 origin/main `b233b7487a02b2435798ca4273efe941bcba34ee` から `codex/23-review-collection-cli` で進めた。PR #22 の統合済み状態と既存 Issue を確認した。ユーザーの指示に従い push・PR 作成・Issue クローズは行わない。

文書一覧、データ・調査仕様の保存と部分失敗・未決定事項、収集仕様の結果と期限、公開デモ仕様の README 要件、操作一覧、開発手順、採用済み ADR-0001・0002・0003・0004・0005 と AGENTS.md を参照した。[CLI 仕様](../specs/review-cli.md) と [ADR-0006](../adr/0006-collection-output-boundary.md) に責務と契約を記録した。保存用 context と writer の制約を Go の公式ドキュメントでも確認した。

## 契約単位の Red・Green

テストだけを追加して Red を確認し、対応する実装だけを追加した。関連テストは固定版 Compose の verify サービスで `go test -count=1` を実行した。同じ契約の入力違い・境界は名前付きテーブルにまとめた。

| 契約 | Red の理由 | Green とコミット |
| --- | --- | --- |
| 保存先を注入する操作 | CollectionSaver と CollectAndSaveReviews が未定義だった | 保存順序と入力不正時の I/O 拒否が成功した。92dbd50 |
| 途中失敗の保存と複合原因 | 収集失敗で保存を呼ばず部分結果を渡さなかった | 最初/途中の失敗でも1回保存し、両方の原因を保持した。0cf5b24 |
| 保存の期限と解放 | 保存に親の context をそのまま渡し、独立した期限と解放がなかった | 値を保持して最大5秒とし、収集キャンセル・期限切れ後の保存と成功/失敗時の解放を確認した。8af1147 |
| JSON の純粋変換 | jsonoutput の実装がなかった | 条件・本文・時刻・ID・空配列・部分結果と schema_version を確認した。0ba0140 |
| JSON の保存 adapter | NewSaver が未定義だった | JSON と改行、書き込み失敗・短い書き込み・キャンセル・期限切れ、Close と再試行を行わないことを確認した。287cf2e |
| 保存形式を知らない CLI | reviewcli の実装がなかった | 引数と境界・help・通信前の拒否、保存先の注入と実行原因の伝播を確認した。7550576 |
| 起動と出力・終了コード | cmd/collect-reviews の run が未定義だった | stdout/stderr、0/1/2、部分結果とキャンセル、出力失敗を確認した。Compose の入口も追加した。981f81b |

実 Client と JSON Saver の httptest 連携、別実装のメモリ Saver、実 CLI プロセスの help・不正入力は初回から Green だった。回帰テストだけの 1c3b76d にまとめ、人工的な Red は作らなかった。重複した保存先ラベルの検査を整理し、JSON 変換の入力保持はコピーと比較した。

最終確認で閉じた stderr の pipe を実プロセスへ渡すテストを追加したところ、`signal: broken pipe` で終了し、終了コード1を返さない Red を確認した。CLI の起動時に SIGPIPE を無視して書き込み失敗を通常のエラーとして扱う修正だけを加え、同じテストが Green となった。SIGINT・SIGTERM のキャンセル処理は維持した。

## 全体検証

`bash scripts/check-app.sh` が終了コード0で完了した。整形・vet・通常テストは cmd/collect-reviews・cmd/server・jsonoutput・reviewcli・steam の5パッケージで成功し、golangci-lint は0件だった。通常テストを含む `go test -tags=integration -race -count=1 ./...` も同じ5パッケージで成功した。CLI のプロセステストはコンテナ内でバイナリをビルドして実行した。

本番用イメージ ID `sha256:74fc2ec8fcaeee2f910e45035718303e606aa16a6b3ab924a45781c29665e3d5` で PORT 8080・10000、HTTP、SIGTERM・SIGINT の正常終了、不正 PORT の非0終了を確認した。本番は既存 HTTP サーバーのみをビルドするため、今回のローカル CLI はそのイメージに含まれない。

SIGPIPE 修正後は `scripts/check-go.sh unit`、`scripts/check-lint.sh`、`scripts/check-go.sh integration` を再実行し、5パッケージの通常テスト・vet・整形、lint 0件、通常テストを含む integration/race の成功を確認した。本番サーバーの入力は変更しておらず、成功済みのイメージ検査は重ねなかった。

README と同じ `docker compose build collect` を実行し、`docker compose run --rm --no-deps -T collect --help` は終了コード0、`--app-id 0` は2と確認した。どちらも stdout は0バイトで stderr に使用方法・診断を出し、CLI まで引数が届いた。これらは実 Steam に接続しない。

各コミットの必須 pre-commit で共通144テスト・actionlint・シェル構文・空白、Go の index 整形を確認し、commit-msg で日本語の本文付きメッセージを確認した。Docker は公開イメージ用の一時的な設定を使い、通常の設定変更やフックの回避を行わなかった。

文書・ネスティング設定の確認は、参照先の存在、表と内容の差分、JSON の構文を確認した。振る舞いを変えない文書・設定に人工的な Red は追加しなかった。

## セルフレビューと未導入項目

AI のセルフレビューで、操作側の保存契約、入口での依存の組み立て、純粋変換と I/O の分離、順序・部分結果・複合エラー・再試行なし、context と接続プール・writer の所有、stdout/stderr、入力と出力の上限を確認した。任意の writer の厳密な終了期限・原子的な保存を保証しないことも仕様へ記載した。

実 Steam へ追加の手動通信を行っていない。通常テストと CI の対象テストは取得依存・httptest を使い、外部 Steam を呼ばない。実 API の連携は前の Issue #17・#21 の少量確認と区別する。

DB の採用と adapter、永続的な取得時刻・本文版・分析集合、再開・再試行、MCP/Web、coverage・分岐・mutation、Render 配置、人によるレビュー、未送信ブランチの GitHub アプリケーション CI は未導入・未実施とする。最後にコミット範囲、差分と作業ツリーを確認し、Issue の完了条件へ結果を対応させる。

## PR 作成後の CI 修正

2026-10-09 のユーザー指示に従いブランチを push し、[Draft PR #24](https://github.com/shigaresso/player-lens/pull/24) を作成した。上記の push 前の記録は当時の操作範囲を示す。

HEAD `b6868f078e8e45cd486733391b2debfd6e0dbbf1` の [アプリケーション CI](https://github.com/shigaresso/player-lens/actions/runs/37798603664) で CLI プロセステストのビルドが `error obtaining VCS status: exit status 128` により失敗した。検証コンテナの読み取り専用 checkout から Git 情報を取得する環境差を確認し、CLI の検証用・ローカル用のビルドに `-buildvcs=false` を指定した。実プロセステストを省略せず、収集や出力の契約は変更しない。修正後のローカル検証と実 CI の結果を PR と Issue に記録する。


## Git 情報の取得に依存しない共通環境への変更

2026-10-09 のユーザー指示に従い、Git 情報を取得できない場合の対応を Issue #23 の範囲・完了条件へ追加した。これまでの `go list`、CLI の検証用・ローカル用ビルド、lint の個別設定を Dockerfile の development ステージの `GOFLAGS=-buildvcs=false` へまとめた。開発・検証・解析・本番ビルドと、テストから起動する子 Go コマンドへ引き継ぐ。Git の safe.directory・ホスト設定は変更しない。

`bash scripts/verify_go_vcs.sh` は隔離した一時モジュールを使い、Git が128を返す場合、Git コマンドが PATH にない場合、`.git` がない場合を同じ契約の状況として検証する。個別フラグを付けていない `go list`・`go build`・`go test` と、テスト内の子 Go ビルドを確認し、一時資源は終了時に削除する。integration の共通検査へ接続し、pre-push と CI の go-integration で継続して実行する。アプリケーションのテストをこの一時モジュールへ置き換えず、既存の integration/race も維持する。

- Red：共通設定のない verify イメージで追加した回帰検査が `error obtaining VCS status: exit status 128` で失敗した。
- Red：回帰検査の終了コード19を注入した Python テストで、integration の共通検査が0を返し、失敗が伝播していなかった。
- Green：共通環境へ移した後、3状況すべてで実 Go コマンドと子ビルドが成功した。Python の対象5テストも成功し、回帰検査の失敗を伝播した。
- `python3 /private/tmp/player-lens-collect-run.py bash scripts/check-app.sh` が終了コード0となった。通常テスト5パッケージ、gofmt・vet・lint 0件、Git 非依存の3状況、既存の integration/race、本番用イメージの HTTP・PORT・SIGTERM/SIGINT・不正 PORT を確認した。本番イメージ ID は `sha256:74fc2ec8fcaeee2f910e45035718303e606aa16a6b3ab924a45781c29665e3d5` となった。

個別フラグを除いた `docker compose run --rm --no-deps --build -T collect --help` も終了コード0となり、CLI の起動を外部通信なしで確認した。

ADR-0001 の共通環境・VCS 情報の制約、ADR-0003 の解析設定、開発手順と CLI 仕様を整合させた。Go バイナリへの VCS revision・時刻・変更状態の自動埋め込みは省略する。対象 SHA と成果物の対応は CI・作業記録とイメージ ID・digest で管理する。Git・GitHub を使うコミット・Issue・PR・Release の規約検査は維持し、取得失敗を成功扱いへ変更しない。共通環境を使わないホスト実行や GOFLAGS を上書きする経路は適用範囲へ含めない。実 Steam へ追加通信は行っていない。
