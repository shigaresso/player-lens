# 開発手順と規約検査

## 導入状態

Issue #3 でコミット・PR・Issue の形式検査、Git の範囲検査、GitHub の読み取り確認、Lefthook、テンプレート、CI の設定を導入した。Issue #5 で Release の事前検査と移植手順、Issue #11 で最小 Go アプリケーション・Docker・アプリケーション検査、Issue #13 で Go リンターとコミット前の整形検査を追加した。DB、coverage・分岐・mutation は未導入とする。

ローカルと GitHub 上の検査結果は、対象 SHA と区別して [Issue #3 の作業記録](worklogs/3-conventions.md) と [Issue #5 の作業記録](worklogs/5-release.md) に残す。PR #4 の規約 CI と、main 導入後の Issue #5 の Issue イベント CI の成功を確認した。PR #4 の統合コミットはメッセージ規約違反で main CI に失敗した。新しいリリース対象では同じ SHA の main CI の成功を要求する。

Release の作成・公開は [Release 手順](releases.md)、他のプロジェクトへの導入は [移植ガイド](reuse.md) に従う。

## ローカルの準備

Python 3.12 以降、Git、`curl`、`tar`、`install` を用意する。ツール配置は macOS arm64 と Linux x86_64 に対応する。

```sh
bash scripts/install-tools.sh
.tools/lefthook install
bash scripts/check.sh
python3 scripts/verify_hooks.py
```

`install-tools.sh` は固定版と SHA-256 を検証して `.tools` に配置する。チェックサムが違う場合は実行ファイルを配置せず失敗する。フックが既にある checkout では、上書き前にその管理方法と用途を確認する。

`check.sh` は規約・開発用ツールの全 Python テスト（0件なら失敗）、未ステージ・ステージ済みの空白、Lefthook 設定、actionlint、シェル構文を確認する。actionlint の ShellCheck 連携は無効にし、シェル構文は `bash -n` で確認する。静的解析による意味の保証とは区別する。

## VS Code のファイルネスティング

リポジトリのルートを VS Code で開くと、`.vscode/settings.json` のワークスペース設定で実装と関連テストをまとめる。通常テストと integration テストは同名の Go 実装の下に、`client.go` と MCP の `collection.go`・`handler.go` の複数の契約テストは同じ親の下に表示する。`go.sum` は存在する場合に `go.mod` の下へまとめる。初期状態は折り畳みとする。

Issue #77 で handler 直下の collect_reviews.go と health.go に関連する契約テストをまとめ、SDK 構築の server.go に通常／結合テストをまとめた。補助処理のサブフォルダを折りたたむと Controller を探せる表示にした。同名の Steam 収集操作のテストも既存の配置で表示できるパターンを維持した。

Python は同じディレクトリの `test_<実装名>.py` を `<実装名>.py` の下へまとめる。たとえば `github.py` の下に `test_github.py` を表示する。`validate.py` は `test_validate.py` と `test_main_range.py` をまとめる。対応する Python の親ファイルがないテストは単独表示とする。

この設定はエクスプローラーの表示を整理する。Go のパッケージ単位のテスト、Python のテスト収集と、実装・テストのファイル構成は維持する。関連テストを追加・改名した場合は、必要に応じて `explorer.fileNesting.patterns` を更新する。公式の設定項目は [ファイルネスティング](https://code.visualstudio.com/updates/v1_67#_explorer-file-nesting) を参照する。

## 各フックの責務

| フック | 検査 |
| --- | --- |
| commit-msg | メッセージファイルを共通検査へ渡す |
| pre-commit | 共通検査と、変更された Go の index の内容の整形検査を実行する。自動修正・再ステージはしない |
| pre-push | stdin の全 ref 更新のコミットを検査してから、規約の共通検査とアプリケーション検査を実行する |

`pre-push` は登録済み remote 名を前提とする。新規 ref は fetch 済みの remote 参照を使って対象を求め、参照がなければ失敗する。既存 ref は remote が提示した旧 SHA から対象を求める。削除はコミットの取得対象にしない。複数 ref のいずれかが違反すれば全体が失敗する。main への直接の更新・削除は拒否する。

空の更新入力は送信対象がない正常な結果とする。Git を使わず直接フックを試す場合も、stdin と remote 名を渡す。`verify_hooks.py` は一時リポジトリでフックを呼び出し、実際の push は行わない。隔離テストではアプリケーション検査と Docker の実行先をテスト用処理へ置き換え、接続と失敗の伝播を確認する。実際の Docker・Go・本番用イメージは次の共通コマンドで別に検証する。

`scripts/staged_format.py` は追加・変更・改名・コピーされた `.go` を index から一時ディレクトリへ取り出す。削除と Go 以外の変更だけなら Docker を使わない。部分ステージや空白付きパスを扱い、index と作業ツリーを保持する。Go の変更がある場合は固定版 verify イメージを用意し、読み取り専用の一時コピーへ gofmt を実行する。整形差分、Docker の停止やビルド失敗はコミットを拒否し、一時コピーは失敗時も解放する。

## Go・Compose・本番用イメージの検査

Docker と Compose 2.23 以降を起動し、[README](../README.md#開発環境) の手順で開発する。`scripts/check.sh` は Docker・ネットワークを使わない Python テストと静的検査を行い、pre-commit のたびに本番用イメージをビルドしない。開発用の起動テストは `python3 -m unittest discover -s scripts -p test_dev.py` でも実行できる。Go の TDD の関連テストは Compose の verify サービスで実行し、変更確定時の必要な Go 検査を別途確認する。pre-push は規約検査の後に `scripts/check-app.sh` を必ず実行する。

```sh
bash scripts/check-app.sh
```

このコマンドは以下を順に実行し、いずれかの失敗で停止する。

- `bash scripts/check-go.sh unit`：読み取り専用ソースの Compose サービスで gofmt の差分を検出し、`go vet ./...` と `go test -count=1 ./...` を実行する。自動修正で成功扱いにしない。
- `bash scripts/check-lint.sh`：読み取り専用ソースの lint サービスで固定版 golangci-lint を実行する。通常のソースと integration タグ付きテストを含める。
- `bash scripts/check-go.sh integration`：実行対象に結合テストが存在することを確認し、`go test -tags=integration -race -count=1 ./...` を実行する。通常のテストも含む。先に `scripts/verify_go_vcs.sh` で Git の失敗・不在と checkout の不在でも Go の処理と子ビルドが成功することを確認する。テスト一覧の取得失敗はテスト未検出とは区別して中断する。外部サービス・DB・実 Steam・実 LLM は使わず、HTTP と終了処理、ローカルの実 SDK クライアントとの MCP 連携を確認する。
- `python3 scripts/check_image.py`：本番用イメージを一度ビルドし、同じ image ID を使って非 root 設定、ソース共有なし・読み取り専用での起動、HTTP・MCP 応答と外部 Host の拒否、PORT の変更、不正設定の非0終了、SIGTERM/SIGINT の正常終了を確認する。Issue #29 の認証なし公開設定でも、同じイメージで初期化・一覧・不正入力・許可 Host・Origin・OAuth metadata の404・不正 URL の起動失敗を確認する。実 Steam への取得は行わない。一時コンテナは削除する。

Docker の停止・未導入でこの検査を省略して成功扱いにしない。Go の coverage 閾値・分岐・mutation 検査は別 Issue で導入するため、このコマンドの成功だけで100%と記載しない。

`.github/workflows/application.yml` に独立した `go-lint`・`go-unit`・`go-integration`・`production-image` のジョブを設定した。PR のソース SHA と main 更新時の SHA を取得し、同じコマンドを実行する。`go-lint` では実ツールの正常・違反の判定も確認する。ローカルの成功と実 CI の成功、PR/main のイメージ検証と配置するリリース成果物の検証を区別し、[Issue #13 の作業記録](worklogs/13-go-lint-format.md) と Issue・PR に対象 SHA の結果を残す。配置用の registry と digest の公開・配置経路は未導入とする。

### Go の整形修正とリンターの更新

全体の整形チェックは `bash scripts/check-go.sh format`、明示的な修正は `bash scripts/check-go.sh format-fix` を使う。修正だけが Compose の format サービスを使い、作業ツリーへ書き込む。修正後に差分を確認してステージする。検査用サービスの読み取り専用設定を変更して修正を行わない。

[ADR-0003](adr/0003-go-format-and-lint.md) に従い、golangci-lint v2.14.0 の公式イメージをタグと複数アーキテクチャ用 index digest で固定し、development と同じ Go 1.27.1 の lint ステージへバイナリを取り込む。設定 `.golangci.yml` は errcheck（blank identifier のエラーも含む）、staticcheck、ineffassign、unused、wsl_v5 を有効にする。検査ツールはアプリケーションの go.mod と本番用イメージへ追加しない。Issue #23 で Dockerfile の development ステージに `GOFLAGS=-buildvcs=false` を共通設定した。開発・検証・解析・本番ビルドと、テスト内で起動する子 Go コマンドがこの設定を引き継ぐ。バイナリの VCS 情報埋め込みは省略し、対象 SHA・成果物の対応は CI と作業記録で管理する。Git・GitHub の規約検査は維持する。Git 非依存の回帰検査は integration の共通コマンドで、pre-push と CI の go-integration から実行する。go-build・go-mod・go-lint volume は再利用可能なキャッシュで、データ永続化ではない。

ツール更新時は版・digest・公式の Go 対応情報を確認し、`python3 scripts/verify_go_tools.py` を実行する。この実 Docker 検証は一時ソースで正常、既存4種類のリンター違反、空行の正常／違反と defer のまとまり、integration タグ、部分ステージ、全体の整形検査と修正を確認し、一時ファイル・コンテナを削除する。通常の Python テストと隔離したフック検証には Docker を要求しない。pre-push はアプリケーション全体の検査を行い、ツール自体の違反入力検証は独立した CI と更新時に行う。

coverage は最初のユースケースが動く段階、mutation はユニットテストが安定した段階で専用 Issue により導入を検討する。AGENTS.md の品質目標を維持し、現在の検査成功を目標達成と記載しない。

### Go の空行規則

Issue #70 で、処理と直後のエラー対応を1セットとして読めるよう、wsl_v5 の err・after-block・after-defer を有効にした。ほかの wsl_v5 の規則は有効にしていない。

- 実行によるエラーの代入と、その直後のエラー確認の間には空行を入れない。
- if・for・switch などのブロックの後に次の文が続く場合は、空行を入れる。
- 資源取得・エラー確認・その資源に対応する defer は隣接を許可する。連続する defer も分断せず、最後の defer と次の処理を空行で区切る。

```go
resource, err := acquire()
if err != nil {
    return err
}
defer resource.Release()

result, err := execute(resource)
if err != nil {
    return err
}

return result, nil
```

このコードは配置の例であり、実際の資源解放のエラー処理は各 API の契約に従う。検査は通常・integration の実装とテストに適用し、pre-push と go-lint CI の読み取り専用 lint で拒否する。pre-commit は従来の index の gofmt 検査を行う。

gofmt と format-fix はこの空行の区切りを自動生成しない。lint の指摘に従って修正し、差分と再検査で確認する。検査中の自動修正・再ステージは行わない。初回の一括修正では別パスの一時コピーだけを修正し、空行以外の行が同じことを確認して反映した。検証結果は docs/worklogs/70-go-error-block-spacing.md に記録する。

機械的な空行検査は、処理の意味や全エラー確認の対応を保証しない。ブロックを持たない文同士の意味上のまとまりはレビューで判断する。スタイルのために処理順序・テストの入力や期待結果を変えたり、関数抽出を行ったりしない。

## 本文検査の契約

- コミットは実在する正の Issue 番号の形式、許可した種別、日本語文字を含む要約、空行、内容のある箇条書きとする。ローカルでは番号の形式を確認し、実在は CI で確認する。
- PR の必須節は「背景・目的」「変更内容」「検証」「参照仕様・ADR」「関連 Issue」とする。Issue の必須節は Issue フォームと同じ8項目とする。
- 必須節は `##` の見出しで記述する。空欄、コメントだけ、コード例だけ、空の箇条書き・チェックリスト、TODO などのプレースホルダーだけでは通らない。見出しの重複も拒否する。
- HTML コメント、fenced code、4スペースまたはタブで始まるコード例を記載内容から除く。Markdown 全体を解釈する処理ではないため、必須項目は通常の段落・箇条書きで記述する。
- Issue の完了条件は内容のあるチェックリストとし、未チェック・チェック済みのどちらも許可する。
- PR の関連番号は「関連 Issue」の節で `Closes #番号` または `Refs #番号` を1行ずつ記載する。コメントや例示の番号は連携として扱わない。
- ラベルは1件以上とし、表示可能な ASCII 文字と英字を含む名前にする。英語として適切か、変更内容に合うかはレビューで確認する。
- 本文・タイトルに文字列として書かれた改行表現を含めない。実際の改行を使用する。

```sh
python3 scripts/validate.py commit /path/to/message.txt
python3 scripts/validate.py range origin/main HEAD
python3 scripts/validate.py pr --title '規約検査を追加した' --body-file docs/worklogs/3-pr.md --label ci
python3 scripts/validate.py issue --title '運用規約を検査する' --body-file /path/to/issue.md --label ci
```

上記の `validate.py` コマンドは外部ネットワークへ接続しない。コミット単体・本文の形式が成功しても、関連番号の実在や内容の正しさを保証しない。main 更新専用の `main-range` は、後述の PR 統合例外を確認するため読み取り API に接続する。

## Issue・PR の作成手順

作成は `github.py` を入口にする。このコマンドは共通の本文検査を内部で呼び出し、GitHub の読み取り確認に成功した後だけ `gh` の作成処理を実行する。作成前の検査コマンドを別途手動で実行する必要はない。

```sh
python3 scripts/github.py issue \
  --repo shigaresso/player-lens \
  --title '対応する内容を日本語で記載する' \
  --body-file /path/to/issue.md --label ci

python3 scripts/github.py pr \
  --repo shigaresso/player-lens \
  --head codex/3-conventions-validation --base main \
  --title '規約検査を追加した' \
  --body-file docs/worklogs/3-pr.md --label ci
```

例の Issue タイトルと本文ファイルは実際の作業内容へ置き換える。`--label` は1件以上必須とし、複数なら繰り返す。作成先の既存ラベルと名前を一致させる。gh のカンマ区切り解釈で検査した名前が変わらないよう、カンマを含む名前は拒否する。不足するラベルは先に別途整備し、このコマンドで自動作成しない。

実行は次の順序とする。

1. 本文ファイルを一度読み、タイトル・本文・ラベルを共通の規約で検査する。失敗した場合、読み取り API と作成処理を呼び出さない。
2. 指定したリポジトリのラベルが存在することを API で確認する。
3. PR は同一リポジトリの送信済み head/base を読み、確定した SHA 同士を比較する。全コミットのメッセージと、本文・コミットから参照した Issue の実在・Issue/PR の区別を確認する。0件、不完全な取得、末尾と HEAD の不一致では失敗する。
4. PR の head/base を再取得し、検査中に変わっていた場合は作成を止める。
5. 検査した本文の同じスナップショットを `--body-file=-` の stdin へ渡して作成する。PR は head/base を明示し、Draft として作成する。入力をシェルコードとして展開せず、編集画面・自動補完で検査済み内容を変更しない。

PR のブランチは先に通常の push 手順で送信する必要がある。未送信のブランチは読み取り API が失敗し、PR を作成しない。`gh pr create` の [head 指定](https://cli.github.com/manual/gh_pr_create) を使って自動 push・fork を抑止する。PR #4 は通常の push 後に、この作成 CLI の検査を通して作成した。

比較 API はページングなしの [250件上限](https://docs.github.com/en/rest/commits/commits#compare-two-commits) を持つため、件数の一致を確認し、取得できた一部だけで成功にしない。250件を超える PR は作成前検査・既存の PR CI とも成功扱いにせず、変更の分割などを検討する。

引数に `--check-only` を追加すると、同じ読み取り検査までを実行し、作成処理を呼び出さない。これは gh の `--dry-run` とは別の実装で、push を呼び出さない。通常の作成ではこのオプションを付けず、内部の事前検査と作成を一度の操作で実行する。

GitHub の認証は gh に委譲する。Codex のサンドボックスから macOS Keychain を利用できない環境では、読み取りを含め、認証が必要なこのコマンドを許可された権限で実行する。認証失敗・通信失敗で検査を省略しない。

作成時のタイムアウトや通信失敗では終了コード1を返し、自動再試行しない。GitHub 側では作成が完了している可能性があるため、既存 Issue・PR の有無を確認してから再実行する。

この入口はブラウザや直接の gh、別クライアントの操作に割り込むものではない。AGENTS.md の作成手順と、作成・編集後の CI を併用する。最後の読み取りと作成を原子的には実行できず、その間のブランチ・ラベルの更新を完全には防げないため、作成後にも CI で現在の状態を確認する。意味の正しさや検証結果の真偽はレビューで確認する。

## GitHub での検査

PR の作成・編集・再開・コミット追加・ラベル変更・レビュー準備で `PR 規約検査 / metadata` を実行する。現在の本文とラベルを API で取得し、同じ SHA の全コミットを検査する。イベント時と現在の HEAD が違う場合や、API のコミット取得が0件・不完全な場合は失敗する。GitHub の PR コミット API は250件の上限があるため、超過しても対象を省略せず失敗する。

コードの共通検査・静的検査・実フック検証は、独立した `規約検査 / verify` で PR 作成・コミット追加・再開、main 更新・手動実行時に行う。本文・タイトルだけの編集とラベル変更では、このワークフローを再実行しない。`PR 規約検査` は固定版ツールの導入・共通テスト・実フックを省略して現在のメタデータを検査する。edited に統合先などの変更を含む場合や変更情報が空・欠落・不正の場合は、PR 規約検査側でも規約の全検査を実行する。未知のイベントや判定処理の失敗は成功扱いにしない。

Draft PR の作成・コミット追加・再開でも、規約の共通検査とアプリケーションの全ジョブを実行する。Draft PR を Ready for review に変更する `ready_for_review` ではメタデータ検査だけを実行し、同じ SHA のコード検査を重ねない。レビュー準備の変更前に、現在の HEAD のコード検査が成功したことを確認する。main 更新時は統合された SHA の規約・アプリケーション検査を実行する。

両ワークフローの concurrency とジョブ名を分け、本文・ラベルの更新がコード検査をキャンセルしたり、失敗した verify を軽量検査の成功で上書きしない。PR では verify・metadata とアプリケーション CI の成功を別々に確認する。メタデータ検査の成功だけをコード検査の成功として扱わない。main の conventions.yml と verify の名前、Release の同じ SHA の push/main 検査は維持する。実行結果は [Issue #15 の作業記録](worklogs/15-pr-metadata-ci.md) と Issue・PR に残す。

PR 本文とコミットで参照した番号を Issue API で確認し、番号が PR を表している場合や、取得できない場合は失敗する。ネットワーク・権限の失敗を成功扱いにしない。Issue イベントでは現在の本文を取得して検査する。GitHub の入力はシェルコードへ埋め込まず、イベント JSON ファイルと `gh` の構造化引数で扱う。

main 更新時は `main-range` で更新前後の全コミットを検査する。通常マージを基本とし、作業コミットを残したまま、確認できた PR 統合の自動生成メッセージだけを独自形式の対象から外す。Squash・Rebase のコミットは通常の書式検査を適用する。初回作成などで旧 SHA が解決できない場合は成功扱いにせず、原因を確認する。

```sh
python3 scripts/validate.py main-range '<更新前 SHA>' '<更新後 SHA>' --repo shigaresso/player-lens
```

この例外は main 更新の検査に限定する。PR CI・作成 CLI・commit-msg・pre-push・通常の `range` は引き続き全コミットの独自形式を検査する。main のテスト・静的検査・実フック検証も維持する。

独自形式を満たさないコミットは、次のすべてを確認できた場合だけ免除する。

- Git の親がちょうど2つである。
- GitHub の関連 PR 一覧をページングして取得し、検査対象 SHA と同じ `merge_commit_sha` を持つ PR が一意にある。
- その PR の詳細でマージ済みと確認でき、統合 SHA が一致する。
- PR の統合先が指定した同一リポジトリの `main` である。
- PR の `head.sha` が統合コミットの第2親と一致する。

PR の対応が欠落・重複・不一致なら免除せず、通常の書式検査を適用する。API の通信・権限・不完全な応答で確認できない場合は失敗する。書式が正常なら免除のための API は呼び出さない。メッセージの先頭が Merge であることだけでは免除しない。main CI は既存の `pull-requests: read` と gh の認証を使い、書き込み権限を追加しない。

GitHub の PR 画面では PR CI・完了条件・差分を確認して `Merge pull request` を選ぶ。GitHub 上で PR 統合との対応を確認する方針であり、署名による生成者の証明や直接 push の防止とは区別する。保護設定の制約は後述のとおりとする。

コード検査の CI は共通テスト・静的検査と実フックの検証も行う。GitHub 上の検査が成功したかは、PR 作成後の run と対象 SHA を確認する。CI・バリデーター自体の変更はセルフレビューで範囲と検査の維持を確認する。

## 必須チェック設定と残る限界

この作業で `main` の保護と Rulesets の API を確認したところ、private リポジトリでは GitHub Pro または公開化が必要という403応答があった。保護は未設定で、フックや CI の設定ファイルだけでマージを機械的に禁止できる状態とは扱わない。

利用可能になったら、次の手順を実施する。

1. PR で `規約検査 / verify`、`PR 規約検査 / metadata` と `アプリケーション検査` の4ジョブ（go-lint・go-unit・go-integration・production-image）が実際に成功したことと、GitHub に表示される必須チェック名を確認する。
2. main の更新を PR 経由にし、このチェックを必須に設定する。
3. 管理者などの迂回範囲も確認し、違反する PR をマージできないことを確かめる。
4. 確認した設定と結果を Issue・PR に記録して導入状態を更新する。

Issue の作成後の検査は、作成そのものを防がない。日本語の自然さ・過去形・Issue の完了・検証結果の真偽・AI が仕様を読んだことは自動保証しない。

## MCP のローカル検証（Issue #25）

公式 Go SDK v1.8.0 を go.mod・go.sum に固定した。Docker は依存取得後にソースをコピーし、Compose Watch はモジュール変更時に再ビルドする。SDK を更新するときは新旧プロトコルの連携、HTTP 制限・キャンセル・終了処理と本番用イメージを再検証する。

通常テストは取得依存を差し替えて入出力と上限を確認する。MCP の接続・一覧・呼び出しは integration タグの実 SDK クライアントとローカル HTTP サーバーで確認する。本番イメージは初期化・一覧・不正入力・外部 Host 拒否を検査し、実 Steam 取得を行わない。ChatGPT のアプリ設定と分析回答の手動確認は [MCP デバッグ](mcp-debug.md) に従い、自動検証と分けて [作業記録](worklogs/25-review-mcp.md) に残す。

## 配置用ツールとワークフロー（Issue #31）

`deploy.py` は main と成功済み CI の検査、一度ビルドした同じ image ID の検証・送信・manifest 照合、digest 固定の Blueprint 生成を分ける。`check_image.py --image` は再ビルドしない。通常テストは API・subprocess・HTTP を差し替え、実 registry・Render・Steam・LLM を呼ばない。新しい Python テストも `check.sh` と必須フックで自動収集する。

`publish-image.yml` は main からの手動起動だけとし、そのジョブだけ packages:write を持つ。既存 CI と Release は読み取り権限を維持する。CI の必要ジョブの成功がない場合に公開を許可する例外は追加しない。外部の状態を変える実行と、ローカル検証・公開 URL の読み取り確認を区別して記録する。

`check_public_mcp.py` の既定は実 Steam を呼ばない。`--collect-app-id` は実取得を行う明示操作であり通常 CI に入れない。[配置手順](deployment.md) と [ADR-0010](adr/0010-render-deployment.md) を参照する。
