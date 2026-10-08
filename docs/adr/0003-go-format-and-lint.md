# ADR-0003：Go の整形・リンターを固定版 Compose で実行する

## 状態と対象

採用。[Issue #13](https://github.com/shigaresso/player-lens/issues/13) の Go 整形、静的解析、フック・CI に適用する。実行結果は [作業記録](../worklogs/13-go-lint-format.md) と PR に残す。

## 背景と比較

gofmt・go vet・テストは導入済みだが、コミット前の Go 整形と未確認エラーなどの検査を追加する。ホストへのツール配置は版がずれやすく、アプリケーションの go.mod への検査ツール追加は実行用の依存へ混入する。固定した公式バイナリを Go の検証用 Docker ステージへ取り込み、共通コマンドから実行する方式を選ぶ。

## 決定

- Go 1.27.1 の development ステージへ golangci-lint v2.14.0 の公式バイナリを取り込む。公式イメージの複数アーキテクチャ用 index digest を `sha256:ad862ba6b3798cbe0fd9fd7408d498fd74fbd2623a92406b2fd3898faf0bf98f` で固定する。本番用ステージへ含めない。
- 実バイナリは Go 1.27.0 でビルドされた v2.14.0 であることを確認した。同じ minor の Go 1.27.1 ソースの正常な解析と違反検出を実行し、版の数字だけで互換性を保証したと扱わない。更新時は公式の対応情報、版・digest、実際の解析をまとめて確認する。
- `.golangci.yml` は `errcheck`、`staticcheck`、`ineffassign`、`unused` を明示し、blank identifier に捨てたエラーも確認する。テストと integration タグを含める。既存の go vet は維持する。通過目的の除外や nolint は追加しない。
- pre-commit は変更された Go の index の内容を一時コピーし、gofmt の差分を拒否する。Go の変更がなければ Docker 検査は省略する。検査で自動修正・再ステージはしない。
- pre-push と独立した `go-lint` CI は同じ検査スクリプトを使う。Issue #23 で `GOFLAGS=-buildvcs=false` を Dockerfile の共通設定へ移し、解析・開発・検証・本番ビルドと子 Go コマンドへ適用する。[ADR-0001 の Git 情報に依存しない Go 実行](0001-runtime-and-deployment.md#git-情報に依存しない-go-実行issue-23) に従い、Git・GitHub の規約検査は維持する。
- 検査サービスのソースは読み取り専用とし、明示的な `format-fix` だけが修正専用サービスの読み書き可能なソースを使う。キャッシュは volume に置く。
- 実ツールの検証は一時ソースで正常・違反と入力の保持を確認し、通常の Python テストや隔離した実フック検証とは分ける。CI でも実ツールの検証を行う。

## 利点と制約

ホスト Go は不要となり、macOS arm64 と CI の Linux amd64 で同じ版を利用できる。Go を変更するコミットには Docker が必要で、初回の取得・ビルドには時間とネットワークが必要となる。並行する修正コマンドと検査は同じソースを変更し得るため、修正後に検査する。

Compose の読み取り専用 mount に CLI で `:rw` を重ねても読み取り専用が残る動作を実検証で確認したため、修正専用サービスを分けた。フックの回避や直接の GitHub 操作を機械的に防ぐものではなく、CI とレビューも維持する。リンターの成功は要求網羅や coverage・mutation の目標達成を意味しない。coverage は最初のユースケースが動く段階、mutation はユニットテストが安定した段階で専用 Issue により検討し、現在の品質目標は維持する。

## 参照

- [公式インストール手順](https://golangci-lint.run/docs/welcome/install/local/)
- [Go バージョンの互換性](https://golangci-lint.run/docs/welcome/faq/)
- [v2.14.0 のリリース](https://github.com/golangci/golangci-lint/releases/tag/v2.14.0)
- [公式の設定形式](https://golangci-lint.run/docs/configuration/file/)
- [開発手順](../development.md)、[ADR-0001](0001-runtime-and-deployment.md)、[ADR-0002](0002-conventions-tooling.md)
