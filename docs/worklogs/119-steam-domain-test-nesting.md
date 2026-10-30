# Steam ドメインの関連テストを実装の下に表示する設定を追加した

[Issue #119](https://github.com/shigaresso/player-lens/issues/119) に対応した。PR #118 統合済みの origin/main（1a39c4f）から codex/119-nest-steam-domain-tests を作成した。

.vscode/settings.json に3つのワイルドカードルールを追加した。collection_state.go は collection_state で始まる6つのテスト、collection_options.go と review_type.go は各2つのテストを親の下へまとめる。既存の設定と折り畳み指定は維持した。テストファイルの分離・配置・内容・実行方法は変更していない。

## 検証

- python3 -m json.tool .vscode/settings.json が成功した。変更前の JSON と比較し、指定した3ルールの追加だけであることを確認した。
- Git 管理下のファイルをディレクトリごとに集め、親パターン・capture の置換・子パターンと実際の名前を静的に照合した。対象10ファイルと既存の collection_failure_test.go・options_test.go の計12ファイルで、親候補がそれぞれ意図した1ファイルだけであることを確認した。
- 変更前後を全フォルダで比較し、新たな親候補が付いたのは steam/domain の7ファイルだけだった。残る3つの同名テストと既存2ファイルの対応は維持し、他フォルダの親候補は変更しなかった。
- VS Code の [公開実装](https://github.com/microsoft/vscode/blob/main/src/vs/workbench/contrib/files/common/explorerFileNestingTrie.ts) を読み、子パターンを親検索へ登録してワイルドカードで照合する処理を確認した。これは公開ソースと設定の静的確認であり、インストール済み VS Code の実 UI を展開・折りたたむ確認は行っていない。
- Go ソース・テスト・go.mod／go.sum・Docker と CI の実行設定が未変更であることを差分で確認した。表示設定と文書だけの変更のため、形式的な Red や新規テストは追加せず、アプリケーション全体の再実行は行っていない。

開発手順の「VS Code のファイルネスティング」に対応と確認手順を追記した。JSON 検査・親候補の静的確認と、実 UI での表示確認を区別した。

## 参照と確認範囲

文書一覧、AGENTS.md の関連処理の配置・テストの1対1対応を一律に強制しない方針、[開発手順](../development.md) の「VS Code のファイルネスティング」「各フックの責務」、ADR-0002 の共有設定と検査の境界を参照した。新しい検査ツールや ADR は追加していない。

AI によるセルフレビューと文書リンク・端末パス非混入の確認を行い、人によるレビューは未実施とした。GitHub 認証と Git 書き込みは許可された権限で実行し、一時的な照合結果と検証ログはリポジトリ外へ置いた。必須フックの結果と確定したコミットは Issue に記録する。

push・PR 作成・Issue のクローズ、GitHub CI、実 Steam／LLM・Render は未実施。coverage 閾値・分岐・mutation（Stryker を含む）は現行では未導入であり、実行済みの検査と区別する。
