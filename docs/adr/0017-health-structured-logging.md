# ADR-0017：生存確認の完了を slog で記録する

## 状態と対象

採用。[Issue #72](https://github.com/shigaresso/player-lens/issues/72) の GET/HEAD /healthz を対象とする。[Issue #74](https://github.com/shigaresso/player-lens/issues/74) で handler 生成と経路登録の境界を更新した。MCP の収集ログや、HTTP 全件のアクセスログは含めない。HTTP の結果・期限・終了と資源の所有は維持する。

## 背景と比較

既存の healthz は本文の書き込み失敗だけをグローバル log へ任意の文字列として出していた。通常の実行結果や時間を調べる記録がなく、テストでグローバル出力先を差し替える必要があった。将来 OpenTelemetry へ接続する予定がある。

- log.Printf の文字列を増やす方式は、項目の検索・集計と context の引き継ぎが困難なため選ばない。
- OTel SDK と保存・転送基盤を今導入する方式は、今回の生存確認ログを超える依存と運用が必要になるため、別の判断とする。
- 標準 slog.Logger を外側で構築して渡し、JSONHandler で stderr へ出す方式を採用する。将来は同じ slog API に otelslog の Handler を接続できる。

## 決定と責務

main が service.name と JSON 出力を設定し、run が serveradapter.NewHealthHandler に logger と時計を渡して handler を生成する。時計はテストで差し替える。独自の logger interface・公開する汎用ログ型・共通サービス層は作らない。

run がルーターも生成し、serveradapter.RegisterRoutes が生成済みの health/MCP handler を既存ルーターへ登録する。以前の NewHandler で health handler の生成を兼ねる構成は、ルーティングへ logger と時計の依存を持ち込むため置き換えた。生成と登録を New と Register で区別し、旧 API の互換ラッパーは残さない。登録中に handler を実行せず、既存の無関係な経路を維持する。

healthHandler は応答と計測、同じ adapter の healthCheckLog は1回の生存確認の結果と固定属性・レベル・安全な原因分類を担当する。関数ごとにイベントを増やさず、health_check.completed の契約を [HTTP 仕様](../specs/http-server.md#生存確認の構造化ログissue-72) にまとめる。

通常 return ごとに1回記録し、LogAttrs へ request context を渡す。将来のトレース情報の経路を維持するもので、仮の trace ID や OTel 型を追加しない。application/domain は変更せず、SDK と送信先・終了時のフラッシュは将来の起動設定の責務とする。

成功は INFO、本文 Write の失敗は ERROR とする。任意のエラー文字列を公開せず、分類不能な原因は unknown とする。既存の原因表示の仕様・テストをこの契約へ引き継ぐ。healthz で選択した200と配信結果を区別し、Write 失敗時の再書き込みは行わない。

起動側の既存ログも server.started・server.failed として同じ JSON logger に揃える。起動失敗の既存診断は保持する。net/http・SDK 自身の全ログを統一する契約は今回扱わない。

## 検証と制約

Go のユニットテストで JSON の項目・1回の出力・応答契約・原因分類・入力の非公開を確認する。制御した時計で duration_ms を確認し、ログの Write エラーでも応答が変わらないこと、通常／キャンセル済みの request context が渡ることを確認する。実 HTTP と実バイナリの起動を結合テストで確認し、race と本番イメージ検証を維持する。

新たな資源・goroutine・バッファは持たず、stdout/stderr の Close は行わない。任意の出力先の非ブロッキングや保存成功は保証しない。保持期間・転送・保存基盤は別の配置判断であり、この導入で永続保存ができたとは扱わない。panic の回復や専用ログも追加しない。

## 参照

- [Go log/slog](https://pkg.go.dev/log/slog)、[otelslog bridge](https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelslog)、[OTel のログモデル](https://opentelemetry.io/docs/specs/otel/logs/data-model/)
- [HTTP 仕様](../specs/http-server.md)、ADR-0001・0003・0013
