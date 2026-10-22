# Issue #38：リクエスト変換の分離

- Issue: https://github.com/shigaresso/player-lens/issues/38
- 起点: fetch 後の origin/main `2ba4223083b1826cc3cdd8bb8e6c234065399d1b`
- ブランチ: `codex/38-steam-request-conversion`。upstream は未設定とし、push は行っていない。

## 変更と参照

`request.go` へ評価種別の数値変換、名前付き API 入力型の組み立て、JSON・クエリ変換を分離した。固定の数値対応は switch にし、不明な評価種別は拒否する。通信資源の管理と公開 API の正規化は維持した。

参照した仕様は `docs/specs/steam-review-page.md` の「API と入力」「結果と失敗」「検証」、データ・調査仕様の「レビューと収集条件」、ADR-0004 の「評価種別の拡張」、ユースケースの「パーツとの境界」「評価種別」。検証環境は開発手順と ADR-0001・0002・0003 に従った。

## 検証

- 変更前: `bash scripts/check-go.sh unit` が全7パッケージで成功した。整形と vet も成功した。
- 変更後: `docker compose run --rm --no-deps -T verify go test -count=1 ./internal/steam ./internal/reviewcli ./internal/jsonoutput` が3パッケージで成功した。
- 既存の送信内容・評価種別・入力拒否・資源解放のテストを維持した。振る舞いを保つ Refactor のため、形式上の Red や実装をなぞるテストは追加しなかった。
- 最終の共通検査とローカル引き渡しの結果は、後続の Issue #40 の作業記録にまとめる。実 CI の実行結果と区別する。

## セルフレビュー

API の JSON キー、パラメーターの順序と値、10秒の期限、非再試行、Body の Close と原因保持の差分を確認した。新しいパッケージ・interface・依存は追加していない。人によるレビューは未実施とする。
