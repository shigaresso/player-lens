# Release の事前検査と公開手順

## 対象と本文

Issue #5 で `scripts/release.py` を導入する。Python 標準ライブラリ、既存の Markdown 検査、読み取り API と gh の認証・書き込みを再利用する。アプリケーションのビルド・配置手順とは区別する。

対象は remote `main` の現在の先端を指す40桁の小文字16進 SHA を必須入力にする。タグは `v0.0.1` のような正式版に限定し、先頭ゼロ、プレリリース、改行を拒否する。タイトルは日本語を含む1行の過去形にする。意味と過去形は内容確認で判断する。

本文は `## 変更内容`、`## 利用方法`、`## 依存環境`、`## 制約`、`## 検証` を必須節とする。[共通の本文検査](development.md#本文検査の契約) と同じく空欄・例示・プレースホルダー・重複見出しを拒否する。実際の改行を使う。

## Draft 作成と公開

1. 対応 PR の差分・完了条件・最新 CI を確認して main にマージする。統合後のコミットメッセージも規約に合わせる。
2. remote main の完全な SHA を取得する。その SHA の push/main の規約 CI が成功してから作成する。
3. 本文ファイルを用意して次の作成コマンドを実行する。検査は作成手順に組み込まれている。
4. GitHub 上の Draft の対象 SHA、タイトル、本文、配布内容を確認する。
5. 同じ対象 SHA で公開コマンドを実行する。保存済み本文を再検査し、公開後もタグと Release を照合する。
6. Release URL、対象 SHA、PR と main の CI の run URL と結果を Issue に記録する。実施済みの完了条件だけチェックする。

```sh
python3 scripts/release.py create \
  --repo shigaresso/player-lens --tag v0.0.1 \
  --target '<remote main の完全な SHA>' \
  --title 'v0.0.1 開発基盤を公開した' \
  --body-file docs/releases/v0.0.1.md

python3 scripts/release.py publish \
  --repo shigaresso/player-lens --tag v0.0.1 \
  --target '<Draft 作成時の完全な SHA>'
```

SHA の説明部分は実際の値に置き換える。どちらにも `--check-only` を追加でき、同じ検査を読み取りだけで実行する。通常は別途の検査コマンドを実行する必要はない。

作成は本文を一度読み、検査した同じ内容を `--notes-file=-` の stdin へ渡す。シェル展開・編集画面・自動生成で差し替えない。`--target` に完全な SHA を渡し、`--draft` を明示する。公開は現在の Draft を読み直し、内容の再確認に成功した後に `--draft=false` に変更する。タグがまだ存在しない Draft も扱い、GitHub が公開時に作成するタグを確認する。

この CLI は追加のバイナリや添付ファイルを受け付けない。今回は GitHub のソースアーカイブを配布する。資産を追加する際は専用 Issue で生成・検証・チェックサム・添付の契約を定める。Release の公開は Draft の解除を意味し、private リポジトリの可視性を変更しない。

## GitHub 上で確認する契約

- 作成時に同名のタグと Release（Draft を含む）があれば失敗する。タグ名が接頭辞として一致する別タグは重複扱いにしない。タグ・Release の一覧はページングして取得する。
- `conventions.yml` の同じ `head_sha`、`event=push`、`branch=main` の最新 run を1件取得し、返された SHA・イベント・ブランチも照合する。`status=completed`、`conclusion=success` を要求する。未実行・途中・失敗・不完全な応答は成功にしない。
- その run のジョブ一覧で `verify` が一意に存在し、完了・成功していることを要求する。スキップを成功扱いにしない。現在の取得上限は100件で、`total_count` と取得件数が異なる場合は対象を省略せず失敗する。
- 公開時は正式版 Draft が一意に存在し、`target_commitish` が明示した SHA と一致することを確認する。`main` などの移動する参照名は受け付けない。
- 既存タグは対象コミットへ解決する。注釈付きタグも追跡し、循環、8段を超える注釈、commit 以外のオブジェクト、別 SHA は失敗する。
- 書き込み直前に main を再取得する。公開では Draft も再取得し、検査中に変わっていれば失敗する。
- 公開後に同名 Release の非 Draft 状態、ID・対象・タイトル・本文・正式版の状態と、タグの対象コミットを照合する。照合が失敗した場合は公開済みの可能性があるため、再公開で解決しようとせず実際の状態を調べる。

認証は gh に委譲する。macOS Keychain の認証がサンドボックスから使えない場合は、[開発手順](development.md#issuepr-の作成手順) に従い許可された権限で実行する。通信・権限の失敗では終了コード1を返し、自動再試行しない。入力が不正なら、公開操作でも API を呼び出す前に停止する。

## 検証と保証の範囲

`scripts/test_release.py` は偽 API と空白を含むパスの偽 gh を使う。Draft 作成・公開・読み取り専用の正常系と、入力違反、CI・ジョブの失敗や欠落、取得件数、対象・タグ・Draft の変化、通信失敗、公開後の不一致を確認する。通常テストは実 GitHub に接続しない。共通の `check.sh` と CI が `test_*.py` を自動収集するため、Release 用のテストも既存の検査で実行する。

最後の読み取りと書き込みは原子的ではなく、その間の変更を完全には防げない。直接の gh・ブラウザ・他クライアントの操作を遮断するものでもない。内容の真偽、移植後の仕様、AI の遵守、リポジトリの不変性はこの検査だけで保証しない。main の保護設定と実装状態は [開発手順](development.md#必須チェック設定と残る限界) を参照する。

## 参照

- [ADR-0002](adr/0002-conventions-tooling.md)
- [移植ガイド](reuse.md)
- [gh release create](https://cli.github.com/manual/gh_release_create)
- [gh release edit](https://cli.github.com/manual/gh_release_edit)
