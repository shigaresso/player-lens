# ADR-0016：MCP の結果処理完了まで同時実行枠を保持する

## 状態と背景

採用。[Issue #62](https://github.com/shigaresso/player-lens/issues/62) に対応する。ADR-0008 の「結果変換完了後に返却する」契約を維持し、ADR-0013 の application と adapter の連携を具体化する。

Issue #56 の分離で、application の defer が adapter の応答変換より先に実行されるようになった。終了後の再受付テストだけでは、応答変換中の早期返却を検出できなかった。利用上限の意味は変更せず、既存の契約を修復する。

## 比較と決定

枠の確保・返却を adapter へ戻す案は、application の受付と実行の責務を分断する。返却関数を結果へ含める案は、呼び出し側の返却忘れや早期返却を許してしまう。SDK の結果変換を application へ移す案は、採用済みの依存の境界に反する。

application の CollectReviews に非 nil の同期的な結果処理関数 func(Result) error を渡す。成功・取得失敗・受付拒否・開始前キャンセルの結果を一度だけ処理し、関数が戻るまで待つ。adapter はこの関数で既存の JSON・エラー分類・応答量判定・SDK 応答生成を行う。application は Result と標準の error だけを扱い、SDK・adapter を import しない。

確保に成功した同時枠だけを application が defer で返す。取得と結果処理の完了・失敗まで同じスコープに含める。結果処理のエラーはそのまま返し、収集の原因は Result.Err に保持して既存の adapter で変換する。拒否時と開始前キャンセルでは確保・返却をしない。開始回数は戻さない。取得用 context は application が解放する。

## 利点と制約

枠の所有者と返却義務を同じ関数に保ち、応答生成の表現を外側に残せる。呼び出し側は結果処理を同期的に完了させ、非同期処理へ逃がさない責務を持つ。HTTP 送信自体の完了待ちは含めない。既存と同じく、context を受け取らない JSON 変換を8秒で強制中断する保証は持たない。

回帰テストで実際の応答変換中の枠を観測し、application の結果処理中・失敗後・拒否時の契約を別に確認する。既存の JSON・SDK 結合・race・本番用イメージ検証も維持する。

## 参照

- [MCP 利用制限](../specs/mcp-usage-limits.md)、[MCP 入出力](../specs/review-mcp.md)、[操作一覧](../use-cases.md)
- ADR-0007・0008・0013。実装と検証は Issue #62 の作業記録に残す。
