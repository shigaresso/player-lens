# Issue #29：認証なしの公開 MCP

## 範囲と履歴

[Issue #29](https://github.com/shigaresso/player-lens/issues/29) の方針を OAuth なしへ更新した。旧 OAuth の9コミットは元ブランチに保護し、最新 origin/main `d0e974dffa0ae303fdb86e65a9b93bfd82283963` から `codex/29-public-mcp-access` を作成した。upstream は設定せず、push・PR 作成・Issue クローズは行わない。

今回は公開 URL と Host・Origin の制御、匿名 MCP と既存共有枠の維持だけを扱う。Web/DB/LLM・共通収集枠への移動・同時上限3への変更は含めない。

## 参照

文書一覧、公開デモ、公開 MCP、レビュー収集 MCP、HTTP サーバー、MCP 利用制限、ユースケース、ADR-0001・0007・0008・0009、開発手順を参照した。

## 検証記録

実装と検証はこれから行う。公開配置・実 ChatGPT 接続は未実施とする。
