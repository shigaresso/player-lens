# ADR-0020：MCP の同時実行枠を収集終了時に返却する

## 状態と背景

採用。[Issue #113](https://github.com/shigaresso/player-lens/issues/113) に対応する。[ADR-0016](0016-mcp-response-slot-lifetime.md) の応答変換完了までの保持と同期 callback の契約を置換する。ADR-0008・0013 の共有状態・確保と返却の所有者・レイヤーの境界は維持し、保持期間だけを変更する。実装・検証の状態は Issue と作業記録で区別する。

ユーザーは、枠の目的を Steam レビュー収集の同時実行制限とした。AI 分析は従来からクライアント側で行い、サーバーの応答変換は JSON・MCP 表現の構築である。旧要件では callback によって応答変換を UseCase の資源保持期間へ含めたが、新しい目的ではこの保持を必要としない。旧実装の不具合修正ではなく、利用制限の対象を変更する判断である。

## 比較と決定

応答変換まで枠を保持し続ける案は、変換中の資源も制限できるが、同期 callback と Controller の外側の変数への代入が残る。callback の出力をジェネリクスで返す案は代入を除けるが、不要になった保持期間と API の複雑さを残す。枠の返却を Controller に渡す案は、所有者を分断して返却忘れを許すため採用しない。

mcpapplication.CollectReviews は callback を受け取らず、既存の Result を直接返す。成功・拒否・開始前キャンセル・取得失敗を Collection・Rejection・Err で表し、同じ原因を別の error 戻り値へ重複させない。確保した同時枠と取得用 context は UseCase 内の defer で、呼び出し元へ戻る前に解放する。収集中の panic も回復せず伝播させ、defer の後始末を維持する。未確保の枠は返却せず、開始回数は戻さない。

Controller は入力解析→UseCase→protocol.EncodeResult と順に呼び出す。応答変換は失敗・panic の場合も含めて枠を使用せず、別の収集を開始できる。Controller の HTTP／SDK context の連携と解放は維持する。application は MCP SDK の型や adapter に依存せず、変換エラーは Controller から SDK へ返す。

## 利点と制約

結果を通常の戻り値で渡し、Controller の流れを順に読める。収集の実行と後始末は UseCase 内で完結する。共有範囲、固定窓、分次・日次の開始回数、8秒の取得期限、入力・部分結果・公開応答を維持する。

応答変換・送信の同時数と CPU・メモリ使用量はこの枠で制限しない。ページごとの Steam API の厳密な QPS 制限でもない。応答量の上限と HTTP の既存の期限は維持し、JSON 変換を context で強制中断する保証は追加しない。

## 検証

収集中は枠を保持し、成功・失敗・キャンセル・期限超過・panic 後は返却し、開始回数と親 context を維持することを確認する。開始前キャンセルと拒否では取得・未確保枠に副作用を与えない。実際の失敗分類処理と4版の SDK の別セッションで、応答変換中の受付を確認する。変換失敗・panic の応答と原因保持・Controller の後始末は担当するテストへ引き継ぐ。

## 参照

- [MCP 利用制限](../specs/mcp-usage-limits.md)、[レビュー収集 MCP](../specs/review-mcp.md)
- [操作一覧](../use-cases.md)、[Controller 一覧](../handler-routing.md)、ADR-0007・0008・0013
