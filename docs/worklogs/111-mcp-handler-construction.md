# MCP HTTP Handler の構築手順を明示した

## 対応範囲

[Issue #111](https://github.com/shigaresso/player-lens/issues/111) に対応した。PR #110 統合済みの origin/main（8f313e3）から codex/111-mcp-handler-construction を作成した。

internal/reviewmcp/adapter/server.go の NewHandler 内で、gate を sharedLimiter、handler を sdkHTTPHandler に改名し、既存の無名関数を createServerForRequest というローカル変数に取り出した。引数の確認、共有 Limiter、Server 生成関数の準備、SDK HTTP Handler、Protect の順序を空行で区切った。新しい委譲先や公開 API は追加していない。

Server 生成関数は SDK に渡し、SDK が必要としたときに実行する。共有 Limiter をその都度作り直さず、従来と同じリクエストの context と依存を newServer へ渡す。nil・ゼロ値の拒否、設定、保護、newServer と Controller の実装を維持した。docs/handler-routing.md にこの構築順序と実行時期を追記した。

## 検証とセルフレビュー

振る舞いを変えない変数名と記述位置の整理のため、形式的な Red や実装の形をなぞる新規テストは追加していない。origin/main の server.go に合意した1ブロックの置換だけを適用すると変更後と一致することを確認した。引数チェック・エラー・newServer の本文・テストは変更していない。

bash scripts/check-app.sh が終了コード0で成功した。

- Docker Compose verify の整形差分検査、go vet ./...、go test -count=1 ./... が成功した。
- 固定版 golangci-lint は通常と integration タグを検査し、0 issuesだった。
- Git の失敗・不在・checkout の不在という3条件で Go の実行と子ビルドが成功した。
- go test -tags=integration -race -count=1 ./... が成功した。通常テストと結合テストの両方を含み、既存の HTTP/MCP、利用枠の共有・保持・返却、キャンセルの契約を確認した。
- 本番用 image ID sha256:157c8828ef478f88ec8da26a7e16d26e601790a9f50678c440009ff099028cce を一度ビルドした。同じイメージで非 root・ソース共有なしの起動、HTTP/MCP・ローカル／公開 Host・Origin、PORT、不正設定の非0終了、SIGTERM／SIGINT の正常終了を確認した。

GitHub 認証・Docker・Git の書き込みは許可された権限で実行した。Docker は既存のソケットと認証情報のない一時 DOCKER_CONFIG を使い、検査の対象・設定・必須フックを緩和していない。必須 pre-commit・commit-msg の結果と確定したコミットは Issue に記録する。

## 参照した仕様・ADR

- [受付とルーティング](../handler-routing.md) の「SDK と middleware の境界」
- [レビュー収集 MCP](../specs/review-mcp.md) の「ツール登録と実行の境界」と結果・失敗・資源の契約
- [データ・調査仕様](../specs/data-and-research.md) の現状と保存・分析の未導入範囲
- [MCP 利用制限](../specs/mcp-usage-limits.md) の共有範囲・設定・受付と固定窓の契約
- ADR-0007・0008・0013・0016 の SDK、共有状態、Controller の配置、応答変換完了までの同時枠の寿命

## 対象外

実 Steam・LLM 接続、Render 配置、GitHub CI、push・PR 作成・Issue のクローズは行っていない。新しい通信や実データの保存先は追加していない。ローカル検証のログはリポジトリ外の一時ディレクトリに置いた。coverage 閾値・分岐網羅・mutation（Stryker を含む）は現行では未導入であり、今回の検証で品質目標の100%達成を保証したとは扱わない。
