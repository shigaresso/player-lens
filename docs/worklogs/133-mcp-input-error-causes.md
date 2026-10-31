# Issue #133：MCP 入力失敗の原因保持と公開情報の分離

## 目的と範囲

入力不正の公開応答を変えず、解析・条件検査から Controller、SDK 結果の非公開エラーまで原因を保持した。実データの保存やログ出力は追加していない。

最新の origin/main（56e704f3f3d8219e3db832126bab89fd116ac69c）から codex/133-mcp-input-error-causes を作成した。push・PR 作成・Issue のクローズは行わない。

## 参照仕様と設計

- MCP 仕様の「接続と入力」「入力変換の境界」「結果・失敗・資源」「結果変換の境界」。入力失敗の公開情報と内部原因の節を追加した。
- 収集仕様の「入力と正規化」「期限と失敗」、データ・調査仕様の現段階の収集と未導入範囲。
- ADR-0007 の SDK と公開応答、ADR-0015 の条件生成、ADR-0020 の Controller・収集・応答変換の寿命。
- 開発手順の Issue 作成、検証と Git フック。

JSON の構文・型エラーを errors.As で、入力不正を errors.Is で識別できるようにした。独自の診断文では未知の項目名・入力値・本文全体を転載せず、既知の項目と拒否理由を説明する。標準 JSON エラーは入力の一部を含むことがあるため、エラーを直接公開・記録する方針にはしない。ログ・OpenTelemetry の収集項目と出力は未導入である。

Controller は ParseInput の原因を InvalidInputResult へ渡す。入力失敗と取得失敗は encodeResultWithCause で公開応答を構築した後、SDK の SetError で内部原因を保持する。SDK v1.8.0 の実装で、本文設定済みなら SetError が本文を変更せず、非公開のエラーが JSON 化されないことを確認した。変換失敗時は既存の固定表示エラーへ両方の原因を保持する。

入力の受理・拒否、大小文字の区別、既定値、公開コード・固定文言・空の Collection・分析ガイドを維持した。不正入力は UseCase を呼ばず、Steam 通信・利用枠を消費しない。context の連携と解放、枠の保持期間・返却、再試行なしの方針は変更していない。

## 契約ごとの Red・Green

1. 入力解析・条件検査の原因保持：TestParseInputFailureCause の12状況を先に追加した。既存実装が同じ入力不正エラーへ置き換えるため、拒否理由の確認で全状況が失敗する Red を確認した。実装後に protocol パッケージの既存テストも含めて Green を確認した。JSON の構文・型、非オブジェクト、未知の項目、null、App ID・件数・ページ数・評価種別・ページ件数・言語の原因を確認した。
2. Controller の原因引き継ぎ：TestCollectInputFailureCause の6状況を先に追加し、result.GetError が nil で失敗する Red を確認した。InvalidInputResult(cause) の契約テストも追加し、引数を受け取る API が未実装のコンパイル失敗を確認した。実装後に handler 以下の関連テストが Green となった。内部原因の保持、固定の公開応答、原因の非公開、通信・枠の非消費を確認した。
3. 実 SDK の公開契約：TestMCPInputFailurePrivacy を追加し、4版それぞれで6種類の不正入力に同一の公開応答を返し、クライアントの GetError が nil であることを確認した。直後の正常呼び出しが1回の利用枠で成功した。これは既存の公開契約の回帰確認であり、最初から Green だったため人工的な Red は作らなかった。

既存の変換失敗のテストは、共通化前の非公開関数ではなく公開の EncodeResult を呼ぶよう機械的に更新し、入力と期待結果は維持した。

## 実行した検証

Go 関連のコマンドは固定版 Docker Compose の verify サービスで実行した。整形には format サービスを使った。

- go test -count=1 ./internal/reviewmcp/adapter/handler/protocol：成功。
- go test -count=1 ./internal/reviewmcp/adapter/handler/...：成功。
- go test -tags=integration -count=1 ./internal/reviewmcp/adapter -run TestMCPInputFailurePrivacy：4版の実 SDK 接続で成功。
- bash scripts/check-go.sh format-fix：整形を適用した。
- bash scripts/check-app.sh：整形、go vet、通常テスト、通常のソースと integration タグ付きテストを含む golangci-lint、Git 非依存の3条件、結合テストを含む race、本番用イメージの全検証が成功した。結合の実行は通常テストも含む。
- 検証した本番用イメージ：sha256:32e568543d7e38588313b17fee3fdb928c036600827feb9502a95d15c558813a。同じイメージで HTTP・MCP、Host・Origin、PORT、SIGTERM・SIGINT、設定不正の終了を確認した。
- 最初のコミットの pre-commit・commit-msg：Gitleaks、規約ツールの184テスト、actionlint・Lefthook・シェル構文・ステージ済み Go の整形、コミット形式が成功した。
- git diff --check：成功。差分をセルフレビューし、入力の公開契約と資源管理を変更していないことを確認した。

## 未実施・制約

coverage 閾値、分岐網羅率、mutation・Stryker は未導入であり、実行していない。実 Steam API・実 ChatGPT・Render への接続、push・PR・CI は実行していない。診断原因を保持したことと、ログを収集・保存できることは区別する。
