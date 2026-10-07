# 開発手順と規約検査

## 導入状態

Issue #3 でコミット・PR・Issue の形式検査、Git の範囲検査、GitHub の読み取り確認、Lefthook、テンプレート、CI の設定を導入した。Issue #5 で Release の事前検査と移植手順、Issue #11 で最小 Go アプリケーション・Docker・アプリケーション検査を追加した。DB、coverage・分岐・mutation は未導入とする。

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

`check.sh` は規約ツールの全テスト（0件なら失敗）、未ステージ・ステージ済みの空白、Lefthook 設定、actionlint、シェル構文を確認する。actionlint の ShellCheck 連携は無効にし、シェル構文は `bash -n` で確認する。静的解析による意味の保証とは区別する。

## 各フックの責務

| フック | 検査 |
| --- | --- |
| commit-msg | メッセージファイルを共通検査へ渡す |
| pre-commit | 共通検査を実行し、ステージ済みの末尾空白などを拒否する |
| pre-push | stdin の全 ref 更新のコミットを検査してから、規約の共通検査とアプリケーション検査を実行する |

`pre-push` は登録済み remote 名を前提とする。新規 ref は fetch 済みの remote 参照を使って対象を求め、参照がなければ失敗する。既存 ref は remote が提示した旧 SHA から対象を求める。削除はコミットの取得対象にしない。複数 ref のいずれかが違反すれば全体が失敗する。main への直接の更新・削除は拒否する。

空の更新入力は送信対象がない正常な結果とする。Git を使わず直接フックを試す場合も、stdin と remote 名を渡す。`verify_hooks.py` は一時リポジトリでフックを呼び出し、実際の push は行わない。アプリケーション検査の呼び出し先は、この隔離テストに限って成功・失敗を制御するテスト用スクリプトへ置き換え、接続と失敗の伝播を確認する。実際の Docker・Go・本番用イメージは次の共通コマンドで別に検証する。

## Go・Compose・本番用イメージの検査

Docker と Compose 2.23 以降を起動し、[README](../README.md#開発環境) の手順で開発する。`scripts/check.sh` は規約ツール専用とし、pre-commit のたびに本番用イメージをビルドしない。TDD の関連テストは Compose の verify サービスで実行し、変更確定時の必要な Go 検査を別途確認する。pre-push は規約検査の後に `scripts/check-app.sh` を必ず実行する。

```sh
bash scripts/check-app.sh
```

このコマンドは以下を順に実行し、いずれかの失敗で停止する。

- `bash scripts/check-go.sh unit`：読み取り専用ソースの Compose サービスで gofmt の差分を検出し、`go vet ./...` と `go test -count=1 ./...` を実行する。自動修正で成功扱いにしない。
- `bash scripts/check-go.sh integration`：実行対象に結合テストが存在することを確認し、`go test -tags=integration -race -count=1 ./...` を実行する。通常のテストも含む。外部ネットワーク・DB・MCP は使わず、HTTP と終了処理を確認する。
- `python3 scripts/check_image.py`：本番用イメージを一度ビルドし、同じ image ID を使って非 root 設定、ソース共有なし・読み取り専用での起動、HTTP 応答、PORT の変更、不正設定の非0終了、SIGTERM/SIGINT の正常終了を確認する。一時コンテナは削除する。

Docker の停止・未導入でこの検査を省略して成功扱いにしない。Go の coverage 閾値・分岐・mutation 検査は別 Issue で導入するため、このコマンドの成功だけで100%と記載しない。

`.github/workflows/application.yml` に独立した `go-unit`・`go-integration`・`production-image` のジョブを設定した。PR のソース SHA と main 更新時の SHA を取得し、同じコマンドを実行する。ローカルの成功と実 CI の成功、PR/main のイメージ検証と配置するリリース成果物の検証を区別する。今回の作業は push しないため、新 CI の実行確認は残る。配置用の registry と digest の公開・配置経路は未導入とする。

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

PR の作成・編集・再開・コミット追加・ラベル変更・レビュー準備で `規約検査 / verify` を実行する。現在の本文とラベルを API で取得し、同じ SHA の全コミットを検査する。イベント時と現在の HEAD が違う場合や、API のコミット取得が0件・不完全な場合は失敗する。GitHub の PR コミット API は250件の上限があるため、超過しても対象を省略せず失敗する。

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

CI は共通テスト・静的検査と実フックの検証も行う。GitHub 上の検査が成功したかは、PR 作成後の run と対象 SHA を確認する。CI・バリデーター自体の変更はセルフレビューで範囲と検査の維持を確認する。

## 必須チェック設定と残る限界

この作業で `main` の保護と Rulesets の API を確認したところ、private リポジトリでは GitHub Pro または公開化が必要という403応答があった。保護は未設定で、フックや CI の設定ファイルだけでマージを機械的に禁止できる状態とは扱わない。

利用可能になったら、次の手順を実施する。

1. PR で `規約検査 / verify` と `アプリケーション検査` の3ジョブが実際に成功したことと、GitHub に表示される必須チェック名を確認する。
2. main の更新を PR 経由にし、このチェックを必須に設定する。
3. 管理者などの迂回範囲も確認し、違反する PR をマージできないことを確かめる。
4. 確認した設定と結果を Issue・PR に記録して導入状態を更新する。

Issue の作成後の検査は、作成そのものを防がない。日本語の自然さ・過去形・Issue の完了・検証結果の真偽・AI が仕様を読んだことは自動保証しない。
