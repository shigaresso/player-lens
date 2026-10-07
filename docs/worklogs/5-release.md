# Issue #5：Release の事前検査と v0.0.1

## 対象と参照

[Issue #5](https://github.com/shigaresso/player-lens/issues/5)。起点は PR #4 の統合済み `origin/main`、`d60968e39290bbfc1f88c3f8c1d86d7f7cf2c8a9`。作業ブランチは `codex/5-release-validation`、upstream は同名の origin ブランチとした。

AGENTS.md の参照義務・Issue・コミット・PR・自動検査、文書一覧、開発手順の本文検査・作成手順・CI、採用済み ADR-0002 の Python・I/O・認証・失敗時の扱いを読んだ。Release の契約を `docs/releases.md`、移植範囲を `docs/reuse.md` に記録した。アプリケーション仕様は変更しなかった。

## Red・Green と実装

状況を1つずつ追加して `python3 -m unittest discover -s scripts -p test_release.py -q` で意図した失敗を確認し、対応する実装だけを変更して Green を確認した。

| 状況 | Red の理由と Green の変更 |
| --- | --- |
| 正常な本文 | release モジュールが未実装。共通 Markdown 検査を接続した |
| 不正なタグ、短い SHA、リポジトリ、文字列の改行 | 対応する拒否条件がなかった。入力検査を状況ごとに追加した |
| Draft 作成 | create が未実装。明示した対象と stdin の本文で Draft を作成した |
| main 不一致、CI 失敗・欠落、必須ジョブのスキップ・不完全取得 | 書き込んだり空の run を参照したりしていた。各条件の読み取りと拒否を追加した |
| 既存タグ・Release、検査中の main 更新、check-only | 書き込みが行われた。重複・再取得・読み取り専用の分岐を追加した |
| Draft 公開、非 Draft、別の対象、欠落 | publish が未実装、または公開後まで拒否が遅れた。保存済み Draft の条件を事前に検査した |
| 既存タグの別 SHA、注釈付きタグ | 公開後だけの拒否、または注釈タグを解決できなかった。事前照合と commit への解決を追加した |
| Draft の同時編集、公開 check-only、公開後の本文変更 | 書き込み、または照合漏れがあった。Draft の再取得と公開後の本文照合を追加した |
| 公開入力の違反 | API が呼び出された。入力検査を通信前に共通化した |
| create/publish の CLI | CLI が未接続、または publish サブコマンドがなかった。通常の入口へ検査を接続した |

共通検査で既に拒否できていた Markdown と、同じ比較で判断できる CI のメタデータ・欠落・通信失敗などは、失敗を作らず契約の回帰テストとして追加した。テストの偽 API の共有では一度、作成テストまで公開用クラスに継承したため、偽 API の共有に限定してテストの記述を修正した。実際の Red を再確認してから実装を進めた。

## ローカルの検証とセルフレビュー

- 共通検査 `bash scripts/check.sh`：規約ツール100テスト、差分の空白、Lefthook 設定、actionlint、シェル構文が成功した。
- `python3 scripts/verify_hooks.py`：隔離した Git リポジトリの実フック5テストが成功した。実 push は行わない検証である。
- Release 通常テストは42件。偽 API と空白を含むパスの偽 gh で、送信内容、読み取り専用、失敗時の書き込み停止、公開後の照合を確認した。実 GitHub の確認とは区別した。
- セルフレビューでは、既存の規約・CI を迂回しないこと、固定した対象とジョブ、ページングと件数、stdin・認証・再試行、Draft と公開の責務、移植後の変更箇所を確認した。人のレビューを実施したとは扱わない。

Go モジュール・本番イメージ・DB・MCP・coverage/mutation の検査は、この変更の対象として未導入である。sql-drill へ書き込みを行わなかった。

## GitHub 上の確認

- [Issue イベント run 37632771541](https://github.com/shigaresso/player-lens/actions/runs/37632771541)：Issue #5、対象 main SHA `d60968e39290bbfc1f88c3f8c1d86d7f7cf2c8a9` で成功した。
- [旧 main の run 37626572602](https://github.com/shigaresso/player-lens/actions/runs/37626572602)：PR #4 の統合メッセージが独自形式・箇条書き・日本語の要約を満たさず失敗していた。履歴を書き換えず、今回の統合メッセージを規約に合わせ、新しい対象 SHA の main CI の成功を要求する。
- 今回の PR・統合後 main の CI、実 Draft と v0.0.1 の公開結果は実行後に Issue #5 と PR に記録する。ローカルの成功だけで実 CI・Release の成功と記載しない。

公開コマンドの利用によってリポジトリを public へ変更しない。GitHub の保護設定も未設定のままとし、未確認の機械的な強制を保証しない。
