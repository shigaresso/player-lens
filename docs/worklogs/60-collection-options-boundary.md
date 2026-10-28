# 収集条件の生成と利用の境界を分離した

## 背景・目的

[Issue #60](https://github.com/shigaresso/player-lens/issues/60) に対応した。同じ公開 struct を未検証の入力と有効な条件に使っていたため、入口での正規化後に収集状態でも再検査していた。収集条件の生成側と利用側を区別し、利用側が有効な値を受け取る契約にした。

最新 origin/main（816743fb9ad7a3d6304a515fbacecabf7f64b88d、PR #59 統合済み）から codex/60-collection-options-boundary を作成した。既存の作業ブランチを維持し、upstream は設定していない。push・PR 作成・Issue のクローズを行わず、ローカルで確認できる状態にする。

## 変更内容

- CollectionOptionsConfig を未検証の設定・表示用コピーとし、CollectionOptions は非公開フィールドに有効な条件を保持する構成にした。NewCollectionOptions が既定値・検査・正規化を担当し、Config はコピーを返す。
- CollectionOptions のゼロ値も有効な既定条件を表し、明示した既定条件の生成値も同じゼロ値に揃えた。生成元・取得したコピーの変更は生成値へ影響しない。生成エラー時もゼロ値を返すため、呼び出し側はエラーを処理する。
- CLI と MCP で条件を生成してから操作へ渡し、CollectionState の条件再検査を除いた。状態生成の App ID 検査、取得依存の検査、期限・キャンセル・解放は維持した。公開 NormalizeCollectionOptions は除き、正規化は生成 API の非公開処理にした。
- Collection.Options は CollectionOptionsConfig のコピーとし、空の Collection の未指定項目と既存 JSON を維持した。MCP の既定20件/3ページ・上限100件/5ページ、厳密な JSON キー・型の検査は変更していない。
- 呼び出し元と fixture を新しい API へ移行し、仕様・操作一覧・ADR-0015 を更新した。AppID・PageOptions・HTTP 入力、他のドメイン型の再設計は含めていない。

## 検証

### Red・Green とコミット

Docker Compose の verify で変更前の `go test -count=1 ./...` が成功した。未検証の型を Config へ改名した段階でも同じ検査が成功した。

| 契約 | 対象テスト | 確認結果 |
| --- | --- | --- |
| 生成 API の既定値・最小最大・不正値・入力非変更・再生成 | TestNewCollectionOptions | NewCollectionOptions と有効な型の未定義による Red 後、生成 API の実装で Green |
| ゼロ値と明示した既定値の意味・表現の一致 | TestCollectionOptionsZero | ゼロ値から未指定項目が返り、生成した既定条件とゼロ値が不一致になる Red 後、固定既定値の取得と表現統一で Green |
| 生成元・取得した設定コピーの独立性 | TestCollectionOptionsConfigCopy | 追加時に既に Green。値のコピーで実装済みであり、形式上の Red は作らなかった |
| 条件生成失敗時の通信・利用枠非消費と空の失敗 JSON | TestCollectionOptionsFailureDoesNotConsumeBudget | 言語・ページ件数の2状況で追加時に Green。プロダクションコードは変更せず、テストだけの fd4381e に分けた |

生成 API・ゼロ値は `go test -count=1 ./internal/steam/domain -run '^対象テスト名$'` で Red を確認し、実装後の domain テストで Green を確認した。コピーも同じパッケージで成功した。生成 API と改名・必要な参照更新・ADR を 911d262 にまとめた。

生成値の受け渡しへの移行後、`go test -count=1 ./...` が成功した。状態・操作・入口・既存テストの API 移行と仕様を ba092c7 に分けた。この段階は動作を維持する Refactor とし、形式上の Red は作っていない。

追加の非消費テストは `go test -count=1 ./internal/reviewmcp/adapter -run '^TestCollectionOptionsFailureDoesNotConsumeBudget$'` と同じ対象の `-race` が成功した。追加後の lint も0 issues で成功した。すべて Docker Compose の verify または既存の検査スクリプトで実行した。

### 不正条件のテストの引き継ぎ

生成済みの型では表現できなくなった不正条件の8状況を、操作・状態から生成 API の既存ケースへ引き継いだ。App ID・依存の拒否は元の操作・状態で維持した。

| 元の契約 | 引き継ぎ先 |
| --- | --- |
| TestCollectReviewsInvalidInput の Japanese・PageSize=101・MaxReviews=10001・MaxPages=101（4状況） | TestNewCollectionOptions の同じ入力値による拒否 |
| TestCollectAndSaveReviewsInvalidInput の MaxReviews=-1（1状況） | TestNewCollectionOptions の収集件数の負数拒否 |
| TestNewCollectionState の Japanese・MaxReviews=10001・MaxPages=101（3状況） | TestNewCollectionOptions の同じ入力値による拒否 |
| 評価種別の有効値と未知の数値 | TestReviewTypeNormalization の収集側を NewCollectionOptions へ移行 |

テストを skip・測定対象外にして通過させていない。外部入力の拒否は TestRunArguments・TestParseInput・TestNormalizeToolInput と既存 MCP 結合で引き続き確認した。生成失敗による不正入力の非消費と空の JSON は追加の回帰テストでも確認した。

### 全体の検証

`bash scripts/check-app.sh` で整形・vet・通常テスト、integration タグを含む lint（0 issues）、Git 非依存の3条件、race 付き通常／結合テスト、本番用イメージ検証が成功した。通常は全15パッケージを収集して14で実行し、cmd/collect-reviews の root は integration タグのプロセステストで検証した。race 付き通常／結合は全15パッケージで成功した。追加のテスト以降はプロダクションコードを変更していない。

本番用 image ID `sha256:f485809492aedd26dda0fd5fba276d1f39b512cf54f2473c1229c97802821678` を一度ビルドし、同じイメージでローカル／公開設定の HTTP・MCP、Host・Origin の拒否、PORT、非 root・読み取り専用・ソース共有なしの起動、不正設定の非0終了、SIGTERM/SIGINT の正常終了を確認した。

必須 pre-commit の規約ツール177テスト・静的検査・index の整形と commit-msg は、各コミットで実行した。ローカルの最終確認には `python3 scripts/validate.py range origin/main HEAD`、`git diff --check origin/main` と PR/Issue 本文の validate.py 検査を使う。PR 本文として使えるこの記録も形式検査の対象にした。

### AI のセルフレビューと実行環境

生成済みの型の全公開値（ゼロ値・生成 API 成功値）が有効であり、設定のコピーを書き換えても内部へ漏れないことを確認した。生成失敗を無視せず拒否する入口、利用枠を確保するまでの順序、結果へコピーを保持する経路を確認した。

既存の JSON・MCP schema・出力変換の実装は変更していない。テスト内の文字列・数値リテラルを origin/main と比較し、差がある既存ファイルは生成テストの拡張・上記8状況の引き継ぎ・追加の非消費テストに限ることを確認した。これをテスト実行の代用とは扱わない。人によるレビューは未実施。

GitHub の認証と Docker はユーザーの指示に従い権限付きで実行した。前回作業の一時 Docker 設定（空の auths と既存 CLI plugin への参照）を再利用し、`DOCKER_CONFIG=/private/tmp/player-lens-58-docker-config DOCKER_HOST=unix://${HOME}/.docker/run/docker.sock` を Docker とフックの実行に付けた。既存の Docker 認証設定は変更せず、検査・フックを省略していない。

### 未実施と保証範囲

push 前のためコード CI と、PR 作成 CLI の送信済み head/base による検査は未実施で、CI の完了待機もしない。実 Steam・LLM・ChatGPT 接続・配置も今回実施していない。coverage 閾値・分岐・mutation（Stryker を含む）は未導入であり、今回の成功をそれらの品質目標達成とは扱わない。

## 参照仕様・ADR

- docs/specs/collect-reviews.md「入力と正規化」「収集状態の境界」「期限と失敗」、steam-review-page.md「API と入力」、data-and-research.md の収集・上限、ADR-0005・0012〜0015。
- docs/specs/review-cli.md の入力・終了・JSON・保存、review-mcp.md「接続と入力」「結果・失敗・資源」、mcp-usage-limits.md「受付と固定窓」、ADR-0006〜0008。
- docs/use-cases.md の境界、docs/development.md の検証・フック・本文検査、ADR-0001〜0003 の実行・検証環境。

## 関連 Issue

Closes #60
