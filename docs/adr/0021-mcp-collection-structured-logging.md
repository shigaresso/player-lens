# ADR-0021：MCP のレビュー収集1回分を slog で記録する

## 状態と背景

採用。Issue #135 で決定した。実装・検証の結果は同 Issue と作業記録に記載する。ADR-0017 の healthz の設計を MCP の Controller に適用し、ADR-0020 の枠と応答生成の寿命は維持する。

収集の失敗原因は保持できるが、実行結果・所要時間・拒否の発生を運用側から確認するログがなかった。将来 OpenTelemetry を導入する予定がある。

## 比較と決定

各関数やページで記録する案は、1回の操作の結果と細部が混ざり、同じ失敗の重複記録を招く。HTTP 全件のアクセスログだけでは SDK 内の collect_reviews と結果を区別できない。今回は Controller の開始から応答生成までを1回のイベントとして記録する。HTTP の送信完了・SDK のディスパッチ前の拒否とは区別する。

main の slog.Logger を run・MCP の組み立てから Controller へ渡す。時計も外から渡し、計測のテストでは制御する。時計は今回の所要時間用であり、既存 limiter の時計と回数の規則は変更しない。構築時に非 nil を確認し、構築時にはログ・計測を実行しない。独自の logger interface やグローバル logger、互換ラッパーを追加しない。

Controller が入力・UseCase・応答生成の流れと結果を所有し、handler/logging の ReviewCollection が固定属性・レベル・安全な分類を担当する。application・domain は観測の出力先を知らない。公開エラーコードは protocol の既存の分類と共有し、応答 JSON を読み戻さない。収集結果と応答生成の結果は別に保持し、本文省略前の件数と返却件数を混同しない。

入力不正・受付拒否・途中失敗も1回記録する。panic は値やスタックを転載せず、固定の分類で記録して回復せず伝播させる。計測用の終了処理は所有 context の解放より先に実行し、処理中の context と値・キャンセル状態をそのまま LogAttrs へ渡す。cleanup が行ったキャンセルを実行失敗と誤認しない。

任意の入力値・本文・カーソル・URL・内部エラー全文を属性へ追加しない。errors.Is・errors.As と固定値で分類し、分類不能な原因は unknown とする。特に JSON の標準エラーは入力の一部を持つため、その Error をログへ渡さない。属性・省略条件・レベルは MCP 仕様で定める。アプリケーション固有の属性を OpenTelemetry の標準属性と称しない。

## 検証と制約

制御した時計とメモリ上の記録先で、1回の出力・context・属性・分類・非漏出を確認する。通常テストは明示した破棄先を渡す。ログ writer の通常の Write エラーでは応答・原因・後始末・枠返却を変更しない。panic と資源の解放は既存テストも維持する。実 SDK の応答とログの対応、実バイナリの stderr を確認し、race と本番用イメージの検証を行う。

新たなファイル・保存先・buffer・goroutine を本番に追加せず、既存の JSON/stderr を使う。Git/Docker に含む実データと ignore の対象追加はない。保存・検索・保持期間・任意の writer の非ブロッキングは保証しない。SDK 自身のログや送信完了の統一、ページごとの計測は対象外とする。

将来は slog の出力設定へ OTel bridge・SDK・exporter と終了時の flush を追加し、必要な span とメトリクスを別途計測する。今回は仮の trace ID や OTel の依存・保存基盤を追加しない。context を渡したことだけでトレース生成・永続保存を実現したとは扱わない。

## 参照

- [レビュー収集 MCP](../specs/review-mcp.md)、[Controller 一覧](../handler-routing.md)
- ADR-0007・0013・0017・0020
- [slog と OTel の公式 bridge](https://pkg.go.dev/go.opentelemetry.io/contrib/bridges/otelslog)
